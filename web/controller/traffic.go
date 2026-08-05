package controller

import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
)

const blimitConfigPath = "/etc/blimit/blimit-config.ini"

type TrafficData struct {
    Installed         bool   `json:"installed"`
    InboundUsed       int64  `json:"inboundUsed"`
    InboundLimit      int64  `json:"inboundLimit"`
    OutboundUsed      int64  `json:"outboundUsed"`
    OutboundLimit     int64  `json:"outboundLimit"`
    InboundThrottled  bool   `json:"inboundThrottled"`
    OutboundThrottled bool   `json:"outboundThrottled"`
    RawConfig         string `json:"rawConfig"`
}

type TrafficController struct{}

func (c *TrafficController) Index(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "traffic.html", gin.H{
        "base_path":   ctx.GetString("base_path"),
        "host":        ctx.Request.Host, // Added for template stability
        "request_uri": ctx.Request.RequestURI,
        "title":       "Traffic Manager",
        "cur_ver":     "1.0.0",
    })
}

func parseConfig() TrafficData {
    data := TrafficData{}
    content, err := os.ReadFile(blimitConfigPath)
    if err != nil {
        return data // Not installed
    }
    data.Installed = true
    data.RawConfig = string(content)

    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "[") || line == "" {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        num, _ := strconv.ParseInt(val, 10, 64)

        switch key {
        case "inbound_bytes":
            data.InboundUsed = num
        case "outbound_bytes":
            data.OutboundUsed = num
        case "inbound_limit_bytes":
            data.InboundLimit = num
        case "outbound_limit_bytes":
            data.OutboundLimit = num
        case "inbound_throttled":
            data.InboundThrottled = num == 1
        case "outbound_throttled":
            data.OutboundThrottled = num == 1
        }
    }
    return data
}

func (c *TrafficController) Status(ctx *gin.Context) {
    ctx.JSON(http.StatusOK, parseConfig())
}

func (c *TrafficController) Install(ctx *gin.Context) {
    cmd := exec.Command("bash", "-c", "curl -sSL https://raw.githubusercontent.com/Tyga-x/pkg_zr1/main/quick-install.sh | bash")
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Install failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Installed successfully"})
}

func (c *TrafficController) Uninstall(ctx *gin.Context) {
    cmd := exec.Command("bash", "-c", "curl -sSL https://raw.githubusercontent.com/Tyga-x/pkg_zr1/main/uninstall-blimit.sh | bash")
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Uninstall failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Uninstalled successfully"})
}

func (c *TrafficController) Save(ctx *gin.Context) {
    var params struct {
        InboundLimit  string `json:"inboundLimit"`
        OutboundLimit string `json:"outboundLimit"`
    }
    if err := ctx.ShouldBind(&params); err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid parameters"})
        return
    }

    content, err := os.ReadFile(blimitConfigPath)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Config not found. Is it installed?"})
        return
    }

    // Use secure pure Go function instead of bash to prevent injection
    inBytesStr, err := convertToBytes(params.InboundLimit)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid Inbound format. Use e.g. 1000GB, 1TB"})
        return
    }
    outBytesStr, err := convertToBytes(params.OutboundLimit)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid Outbound format. Use e.g. 1000GB, 1TB"})
        return
    }

    lines := strings.Split(string(content), "\n")
    for i, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "inbound_limit_bytes=") {
            lines[i] = "inbound_limit_bytes=" + inBytesStr
        } else if strings.HasPrefix(line, "outbound_limit_bytes=") {
            lines[i] = "outbound_limit_bytes=" + outBytesStr
        } else if strings.HasPrefix(line, "inbound_throttled=") {
            lines[i] = "inbound_throttled=0" // Reset throttle status on save
        } else if strings.HasPrefix(line, "outbound_throttled=") {
            lines[i] = "outbound_throttled=0"
        }
    }

    err = os.WriteFile(blimitConfigPath, []byte(strings.Join(lines, "\n")), 0644)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Failed to write config"})
        return
    }

    // Clear existing tc rules so the daemon starts fresh with the new limits
    resetScript := `
        CONF="/etc/blimit/blimit-config.ini"
        IFACE=$(grep interface $CONF | cut -d= -f2)
        tc qdisc del dev $IFACE root 2>/dev/null
        tc qdisc del dev $IFACE ingress 2>/dev/null
        tc qdisc del dev ifb0 root 2>/dev/null
    `
    exec.Command("bash", "-c", resetScript).Run()

    // Restart service to apply changes immediately
    exec.Command("systemctl", "restart", "blimit-monitor").Run()

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Settings saved"})
}

func (c *TrafficController) Reset(ctx *gin.Context) {
    resetScript := `
        CONF="/etc/blimit/blimit-config.ini"
        IFACE=$(grep interface $CONF | cut -d= -f2)
        sed -i "s/inbound_throttled=.*/inbound_throttled=0/" $CONF
        sed -i "s/outbound_throttled=.*/outbound_throttled=0/" $CONF
        sed -i "s/offset_inbound=.*/offset_inbound=0/" $CONF
        sed -i "s/offset_outbound=.*/offset_outbound=0/" $CONF
        sed -i "s/inbound_bytes=.*/inbound_bytes=0/" $CONF
        sed -i "s/outbound_bytes=.*/outbound_bytes=0/" $CONF
        tc qdisc del dev $IFACE root 2>/dev/null
        tc qdisc del dev $IFACE ingress 2>/dev/null
        tc qdisc del dev ifb0 root 2>/dev/null
        systemctl restart blimit-monitor
    `
    cmd := exec.Command("bash", "-c", resetScript)
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Reset failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Usage reset"})
}

// Helper to convert e.g. "1GB" to bytes using pure Go (Safe from bash injection)
func convertToBytes(input string) (string, error) {
    input = strings.TrimSpace(strings.ToUpper(input))
    if input == "" || input == "0" {
        return "0", nil
    }

    var multiplier int64
    var numStr string

    // Check suffix and set multiplier
    if strings.HasSuffix(input, "TB") {
        multiplier = 1024 * 1024 * 1024 * 1024
        numStr = strings.TrimSuffix(input, "TB")
    } else if strings.HasSuffix(input, "GB") {
        multiplier = 1024 * 1024 * 1024
        numStr = strings.TrimSuffix(input, "GB")
    } else if strings.HasSuffix(input, "MB") {
        multiplier = 1024 * 1024
        numStr = strings.TrimSuffix(input, "MB")
    } else {
        return "0", fmt.Errorf("invalid format (use TB, GB, or MB)")
    }

    // Parse the number part
    num, err := strconv.ParseFloat(numStr, 64)
    if err != nil {
        return "0", fmt.Errorf("invalid number format")
    }

    // Calculate total bytes
    bytes := int64(num * float64(multiplier))
    return strconv.FormatInt(bytes, 10), nil
}