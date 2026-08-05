package controller

import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "regexp"
    "strings"

    "github.com/gin-gonic/gin"
)

type TorrentController struct{}

const trackersFile = "/etc/trackers"
const hostsTrackersFile = "/etc/hostsTrackers"

func (c *TorrentController) Index(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "torrent.html", gin.H{
        "base_path":   ctx.GetString("base_path"),
        "host":        ctx.Request.Host,
        "request_uri": ctx.Request.RequestURI,
        "title":       "Torrent Blocker",
        "cur_ver":     "1.0.0",
    })
}

func (c *TorrentController) Status(ctx *gin.Context) {
    installed := false
    var trackers []string
    
    // Check if the tool is installed by looking for the config file
    if _, err := os.Stat(trackersFile); err == nil {
        installed = true
        content, _ := os.ReadFile(trackersFile)
        lines := strings.Split(string(content), "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if line != "" {
                trackers = append(trackers, line)
            }
        }
    }
    
    ctx.JSON(http.StatusOK, gin.H{
        "success": true,
        "obj": gin.H{
            "installed": installed,
            "trackers":  trackers,
        },
    })
}

func (c *TorrentController) Install(ctx *gin.Context) {
    // Using curl to install, exactly like your original tool
    cmd := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/Mishawo/block-publictorrent-iptables/main/ptbinsta.sh | bash")
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Install failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Installed successfully"})
}

func (c *TorrentController) Uninstall(ctx *gin.Context) {
    // Cleaned up the uninstall command to use curl for consistency
    cmd := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/Mishawo/block-publictorrent-iptables/main/uninstall_all.sh | bash")
    err := cmd.Run()
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Uninstall failed: " + err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Uninstalled successfully"})
}

func (c *TorrentController) AddTracker(ctx *gin.Context) {
    var params struct {
        Entry string `json:"entry"`
    }
    if err := ctx.ShouldBind(&params); err != nil || params.Entry == "" {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid entry"})
        return
    }

    entry := strings.TrimSpace(params.Entry)

    // Basic sanitization to prevent bash injection
    if !regexp.MustCompile(`^[a-zA-Z0-9\.\-:]+$`).MatchString(entry) {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid characters in entry"})
        return
    }

    // Check if already exists
    content, _ := os.ReadFile(trackersFile)
    if strings.Contains(string(content), entry) {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Entry already exists"})
        return
    }

    // Append to /etc/trackers
    f, err := os.OpenFile(trackersFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
    if err != nil {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Cannot open trackers file. Is it installed?"})
        return
    }
    f.WriteString(entry + "\n")
    f.Close()

    // Append to /etc/hostsTrackers
    f2, _ := os.OpenFile(hostsTrackersFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
    if f2 != nil {
        f2.WriteString(entry + "\n")
        f2.Close()
    }

    // Run a bash command to apply iptables and hosts rules immediately
    // This now perfectly matches the logic in your bmenu.sh script
    applyScript := fmt.Sprintf(`
        HOST="%s"
        HOSTS_FILE="/etc/hosts"
        
        # Add to /etc/hosts if it's a domain (not an IP)
        if ! [[ "$HOST" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] && ! [[ "$HOST" =~ : ]]; then
            grep -Eq "^[[:space:]]*(0\.0\.0\.0|127\.0\.0\.1|::1)[[:space:]]+$HOST([[:space:]]|$)" "$HOSTS_FILE" || echo "0.0.0.0 $HOST" >> "$HOSTS_FILE"
        fi

        # Resolve domain to IPs
        ips=$(getent ahosts "$HOST" 2>/dev/null | awk '{print $1}' | sort -u)
        for ip in $ips; do
            cmd="iptables"
            if [[ "$ip" =~ : ]] && command -v ip6tables >/dev/null; then cmd="ip6tables"; fi
            
            # Check if rule exists before adding to prevent duplicates
            $cmd -w 2 -C INPUT -s "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A INPUT -s "$ip" -j DROP
            $cmd -w 2 -C OUTPUT -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A OUTPUT -d "$ip" -j DROP
            $cmd -w 2 -C FORWARD -s "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A FORWARD -s "$ip" -j DROP
            $cmd -w 2 -C FORWARD -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A FORWARD -d "$ip" -j DROP
            
            # Block Docker traffic if the chain exists (crucial for VPS with Docker)
            if $cmd -S DOCKER-USER >/dev/null 2>&1; then
                $cmd -w 2 -C DOCKER-USER -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A DOCKER-USER -d "$ip" -j DROP
            fi
        done
        
        # Save rules so they persist after reboot
        if command -v iptables-save >/dev/null 2>&1; then
            mkdir -p /etc/iptables
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
        fi
        if command -v ip6tables-save >/dev/null 2>&1; then
            mkdir -p /etc/iptables
            ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true
        fi
    `, entry)
    
    exec.Command("bash", "-c", applyScript).Run()

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Tracker added and blocked"})
}

func (c *TorrentController) RemoveTracker(ctx *gin.Context) {
    var params struct {
        Entry string `json:"entry"`
    }
    if err := ctx.ShouldBind(&params); err != nil || params.Entry == "" {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid entry"})
        return
    }

    entry := strings.TrimSpace(params.Entry)
    if !regexp.MustCompile(`^[a-zA-Z0-9\.\-:]+$`).MatchString(entry) {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid characters in entry"})
        return
    }

    // Remove from /etc/trackers and /etc/hostsTrackers
    for _, file := range []string{trackersFile, hostsTrackersFile} {
        if content, err := os.ReadFile(file); err == nil {
            lines := strings.Split(string(content), "\n")
            var newLines []string
            for _, line := range lines {
                if strings.TrimSpace(line) != entry {
                    newLines = append(newLines, line)
                }
            }
            os.WriteFile(file, []byte(strings.Join(newLines, "\n")), 0644)
        }
    }

    // Run a bash command to remove iptables and hosts rules
    applyScript := fmt.Sprintf(`
        HOST="%s"
        HOSTS_FILE="/etc/hosts"
        
        # Remove from /etc/hosts
        sed -i.bak -E "/^[[:space:]]*(0\.0\.0\.0|127\.0\.0\.1|::1)[[:space:]]+${HOST//\./\\.}([[:space:]]|$)/d" "$HOSTS_FILE"
        
        # Resolve domain to IPs
        ips=$(getent ahosts "$HOST" 2>/dev/null | awk '{print $1}' | sort -u)
        for ip in $ips; do
            cmd="iptables"
            if [[ "$ip" =~ : ]] && command -v ip6tables >/dev/null; then cmd="ip6tables"; fi
            
            # Loop to delete all instances of the rule
            while $cmd -w 2 -C INPUT -s "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D INPUT -s "$ip" -j DROP; done
            while $cmd -w 2 -C OUTPUT -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D OUTPUT -d "$ip" -j DROP; done
            while $cmd -w 2 -C FORWARD -s "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D FORWARD -s "$ip" -j DROP; done
            while $cmd -w 2 -C FORWARD -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D FORWARD -d "$ip" -j DROP; done
            
            # Remove from Docker chain if it exists
            if $cmd -S DOCKER-USER >/dev/null 2>&1; then
                while $cmd -w 2 -C DOCKER-USER -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D DOCKER-USER -d "$ip" -j DROP; done
            fi
        done
        
        # Save rules so they persist after reboot
        if command -v iptables-save >/dev/null 2>&1; then
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
        fi
        if command -v ip6tables-save >/dev/null 2>&1; then
            ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true
        fi
    `, entry)
    
    exec.Command("bash", "-c", applyScript).Run()

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Tracker removed and unblocked"})
}