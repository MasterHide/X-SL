package controller

import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"

    "x-ui/logger" // Make sure to import your logger
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
        "host":        ctx.Request.Host,
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
    data := parseConfig()
    ctx.JSON(http.StatusOK, gin.H{
        "success": true,
        "obj":     data,
    })
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
        InboundLimit  string `json:"inboundLimit" form:"inboundLimit"`
        OutboundLimit string `json:"outboundLimit" form:"outboundLimit"`
    }
    
    // FIX: Use ShouldBind instead of ShouldBindJSON to accept all content types
    if err := ctx.ShouldBind(&params); err != nil {
        logger.Error("Traffic Save Bind Error:", err)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid parameters sent: " + err.Error()})
        return
    }

    content, err := os.ReadFile(blimitConfigPath)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Config not found. Is it installed?"})
        return
    }

    // Convert inputs to bytes
    inBytesStr, err := convertToBytes(params.InboundLimit)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid Inbound format: " + err.Error()})
        return
    }
    outBytesStr, err := convertToBytes(params.OutboundLimit)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid Outbound format: " + err.Error()})
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
            lines[i] = "inbound_throttled=0" 
        } else if strings.HasPrefix(line, "outbound_throttled=") {
            lines[i] = "outbound_throttled=0"
        }
    }

    err = os.WriteFile(blimitConfigPath, []byte(strings.Join(lines, "\n")), 0644)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Failed to write config"})
        return
    }

    // Clear existing tc rules and restart service to apply immediately
    resetScript := `
        CONF="/etc/blimit/blimit-config.ini"
        IFACE=$(grep interface $CONF | cut -d= -f2)
        if [ -n "$IFACE" ]; then
            tc qdisc del dev $IFACE root 2>/dev/null
            tc qdisc del dev $IFACE ingress 2>/dev/null
        fi
        tc qdisc del dev ifb0 root 2>/dev/null
        systemctl restart blimit-monitor
    `
    cmd := exec.Command("bash", "-c", resetScript)
    out, err := cmd.CombinedOutput()
    if err != nil {
        // Log the error but don't fail the whole request, as the config was saved successfully
        logger.Error("Traffic Save Apply Error:", string(out), err)
        ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Settings saved, but failed to restart service."})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Settings saved and applied successfully"})
}

func (c *TrafficController) Reset(ctx *gin.Context) {
    // This script resets usage to 0 AND removes the quota limits (sets to 0)
    resetScript := `
        CONF="/etc/blimit/blimit-config.ini"
        IFACE=$(grep interface $CONF | cut -d= -f2)
        
        # Reset Usage Counters
        sed -i "s/inbound_bytes=.*/inbound_bytes=0/" $CONF
        sed -i "s/outbound_bytes=.*/outbound_bytes=0/" $CONF
        sed -i "s/offset_inbound=.*/offset_inbound=0/" $CONF
        sed -i "s/offset_outbound=.*/offset_outbound=0/" $CONF
        
        # Reset Throttle Status
        sed -i "s/inbound_throttled=.*/inbound_throttled=0/" $CONF
        sed -i "s/outbound_throttled=.*/outbound_throttled=0/" $CONF
        
        # Clear Quota Limits (Set to 0 = Unlimited)
        sed -i "s/inbound_limit_bytes=.*/inbound_limit_bytes=0/" $CONF
        sed -i "s/outbound_limit_bytes=.*/outbound_limit_bytes=0/" $CONF
        
        # Remove active TC rules
        if [ -n "$IFACE" ]; then
            tc qdisc del dev $IFACE root 2>/dev/null
            tc qdisc del dev $IFACE ingress 2>/dev/null
        fi
        tc qdisc del dev ifb0 root 2>/dev/null
        
        # Restart daemon to apply changes
        systemctl restart blimit-monitor
    `
    cmd := exec.Command("bash", "-c", resetScript)
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Reset failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Usage and limits reset successfully"})
}

// Improved robust conversion (handles bare numbers, KB, MB, GB, TB)
func convertToBytes(input string) (string, error) {
    input = strings.TrimSpace(strings.ToUpper(input))
    if input == "" || input == "0" {
        return "0", nil
    }

    var multiplier int64 = 1
    var numStr string

    if strings.HasSuffix(input, "TB") {
        multiplier = 1024 * 1024 * 1024 * 1024
        numStr = strings.TrimSuffix(input, "TB")
    } else if strings.HasSuffix(input, "GB") {
        multiplier = 1024 * 1024 * 1024
        numStr = strings.TrimSuffix(input, "GB")
    } else if strings.HasSuffix(input, "MB") {
        multiplier = 1024 * 1024
        numStr = strings.TrimSuffix(input, "MB")
    } else if strings.HasSuffix(input, "KB") {
        multiplier = 1024
        numStr = strings.TrimSuffix(input, "KB")
    } else {
        // If no suffix, assume it's already in bytes
        numStr = input
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
