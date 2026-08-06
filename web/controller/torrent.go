package controller

import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "regexp"
    "strings"

    "x-ui/logger"
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
    // FIX: Download script first, strip the interactive bmenu launch, then run.
    // The curl|bash pattern does NOT pass stdin to the inner script,
    // causing bmenu's interactive read() to loop forever with "Invalid option".
    // The Go backend handles add/remove directly — no need for bmenu during install.
    script := `
export TERM=xterm
export SUDO_USER=root
set -e

echo "[+] Downloading install script..."
curl -fsSL https://raw.githubusercontent.com/Mishawo/block-publictorrent-iptables/main/ptbinsta.sh -o /tmp/ptbinsta.sh

echo "[+] Removing interactive bmenu launch (backend handles add/remove directly)..."
sed -i '/bmenu/d' /tmp/ptbinsta.sh

chmod +x /tmp/ptbinsta.sh

echo "[+] Running installer (non-interactive)..."
printf '4\n4\n4\n4\n4\n' | bash /tmp/ptbinsta.sh || true

rm -f /tmp/ptbinsta.sh
echo "[+] Install completed successfully."
`

    cmd := exec.Command("bash", "-c", script)
    cmd.Env = append(os.Environ(), "SUDO_USER=root", "TERM=xterm")
    out, err := cmd.CombinedOutput()
    if err != nil {
        errMsg := fmt.Sprintf("Install failed: %v\nOutput: %s", err, string(out))
        logger.Error("Torrent Install Error:", errMsg)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": errMsg})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Installed successfully"})
}

func (c *TorrentController) Uninstall(ctx *gin.Context) {
    cmd := exec.Command("bash", "-c", "export TERM=xterm; export SUDO_USER=root; curl -fsSL https://raw.githubusercontent.com/Mishawo/block-publictorrent-iptables/main/uninstall_all.sh | bash")
    cmd.Env = append(os.Environ(), "SUDO_USER=root", "TERM=xterm")

    out, err := cmd.CombinedOutput()
    if err != nil {
        errMsg := fmt.Sprintf("Uninstall failed: %v\nOutput: %s", err, string(out))
        logger.Error("Torrent Uninstall Error:", errMsg)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": errMsg})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Uninstalled successfully"})
}

func (c *TorrentController) AddTracker(ctx *gin.Context) {
    var params struct {
        Entry string `json:"entry" form:"entry"`
    }

    // FIX: Try ShouldBind first (auto-detects JSON or form).
    // If that fails or Entry is empty, fall back to ctx.PostForm.
    ctx.ShouldBind(&params)

    entry := strings.TrimSpace(params.Entry)
    if entry == "" {
        entry = strings.TrimSpace(ctx.PostForm("entry"))
    }
    if entry == "" {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid entry: no value provided"})
        return
    }

    // Basic sanitization to prevent bash injection
    if !regexp.MustCompile(`^[a-zA-Z0-9\.\-:]+$`).MatchString(entry) {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid characters in entry. Only letters, numbers, dots, hyphens, and colons are allowed."})
        return
    }

    // Check if already exists
    content, _ := os.ReadFile(trackersFile)
    if strings.Contains(string(content), entry) {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Entry already exists in block list"})
        return
    }

    // Append to /etc/trackers
    f, err := os.OpenFile(trackersFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
    if err != nil {
        logger.Error("Torrent AddTracker: cannot open trackers file:", err)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Cannot open trackers file. Is the tool installed?"})
        return
    }
    f.WriteString(entry + "\n")
    f.Close()

    // Append to /etc/hostsTrackers
    f2, err2 := os.OpenFile(hostsTrackersFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
    if err2 == nil {
        f2.WriteString(entry + "\n")
        f2.Close()
    }

    // Apply iptables + hosts rules using environment variable for security
    applyScript := `
export TERM=xterm
HOST="$XSL_ENTRY"
HOSTS_FILE="/etc/hosts"

# Add to /etc/hosts if it's a domain (not an IP)
if ! [[ "$HOST" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] && ! [[ "$HOST" =~ : ]]; then
    if ! grep -Eq "^[[:space:]]*(0\.0\.0\.0|127\.0\.0\.1|::1)[[:space:]]+${HOST//\./\\.}([[:space:]]|$)" "$HOSTS_FILE"; then
        echo "0.0.0.0 $HOST" >> "$HOSTS_FILE"
        echo "Added $HOST to /etc/hosts"
    fi
fi

# Resolve domain to IPs
ips=$(getent ahosts "$HOST" 2>/dev/null | awk '{print $1}' | sort -u)
if [ -z "$ips" ]; then
    echo "WARNING: Could not resolve $HOST to IP addresses right now (domain/hosts block still active)."
    # Still save and exit cleanly
    if command -v iptables-save >/dev/null 2>&1; then
        mkdir -p /etc/iptables
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
    fi
    exit 0
fi

for ip in $ips; do
    [ -z "$ip" ] && continue
    cmd="iptables"
    if [[ "$ip" =~ : ]] && command -v ip6tables >/dev/null 2>&1; then
        cmd="ip6tables"
    fi

    # Add rules if they don't exist
    $cmd -w 2 -C INPUT -s "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A INPUT -s "$ip" -j DROP
    $cmd -w 2 -C OUTPUT -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A OUTPUT -d "$ip" -j DROP
    $cmd -w 2 -C FORWARD -s "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A FORWARD -s "$ip" -j DROP
    $cmd -w 2 -C FORWARD -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A FORWARD -d "$ip" -j DROP

    # Docker chain if present
    if $cmd -S DOCKER-USER >/dev/null 2>&1; then
        $cmd -w 2 -C DOCKER-USER -d "$ip" -j DROP 2>/dev/null || $cmd -w 2 -A DOCKER-USER -d "$ip" -j DROP
    fi

    echo "Blocked IP: $ip"
done

# Save iptables rules
if command -v iptables-save >/dev/null 2>&1; then
    mkdir -p /etc/iptables
    iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
fi
if command -v ip6tables-save >/dev/null 2>&1; then
    mkdir -p /etc/iptables
    ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true
fi

# Restart netfilter persistence if available
if systemctl list-unit-files 2>/dev/null | grep -q 'netfilter-persistent'; then
    systemctl restart netfilter-persistent 2>/dev/null || true
fi

echo "Done."
`

    cmd := exec.Command("bash", "-c", applyScript)
    cmd.Env = append(os.Environ(), "XSL_ENTRY="+entry, "TERM=xterm")
    out, err := cmd.CombinedOutput()
    if err != nil {
        logger.Error("Torrent AddTracker Apply Error:", string(out), err)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Entry added to list but iptables apply failed: " + string(out)})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Tracker added and blocked successfully"})
}

func (c *TorrentController) RemoveTracker(ctx *gin.Context) {
    var params struct {
        Entry string `json:"entry" form:"entry"`
    }

    // FIX: Robust binding with fallback
    ctx.ShouldBind(&params)

    entry := strings.TrimSpace(params.Entry)
    if entry == "" {
        entry = strings.TrimSpace(ctx.PostForm("entry"))
    }
    if entry == "" {
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Invalid entry: no value provided"})
        return
    }

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

    // Apply removal using environment variable for security
    applyScript := `
export TERM=xterm
HOST="$XSL_ENTRY"
HOSTS_FILE="/etc/hosts"

# Remove from /etc/hosts
sed -i.bak -E "/^[[:space:]]*(0\.0\.0\.0|127\.0\.0\.1|::1)[[:space:]]+${HOST//\./\\.}([[:space:]]|$)/d" "$HOSTS_FILE" 2>/dev/null || true

# Resolve domain to IPs
ips=$(getent ahosts "$HOST" 2>/dev/null | awk '{print $1}' | sort -u)
if [ -z "$ips" ]; then
    echo "WARNING: Could not resolve $HOST — removed list/hosts entries only."
    if command -v iptables-save >/dev/null 2>&1; then
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
    fi
    exit 0
fi

for ip in $ips; do
    [ -z "$ip" ] && continue
    cmd="iptables"
    if [[ "$ip" =~ : ]] && command -v ip6tables >/dev/null 2>&1; then
        cmd="ip6tables"
    fi

    # Remove all matching rules
    while $cmd -w 2 -C INPUT -s "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D INPUT -s "$ip" -j DROP; done
    while $cmd -w 2 -C OUTPUT -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D OUTPUT -d "$ip" -j DROP; done
    while $cmd -w 2 -C FORWARD -s "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D FORWARD -s "$ip" -j DROP; done
    while $cmd -w 2 -C FORWARD -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D FORWARD -d "$ip" -j DROP; done

    if $cmd -S DOCKER-USER >/dev/null 2>&1; then
        while $cmd -w 2 -C DOCKER-USER -d "$ip" -j DROP 2>/dev/null; do $cmd -w 2 -D DOCKER-USER -d "$ip" -j DROP; done
    fi

    echo "Unblocked IP: $ip"
done

# Save iptables rules
if command -v iptables-save >/dev/null 2>&1; then
    iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
fi
if command -v ip6tables-save >/dev/null 2>&1; then
    ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true
fi

# Restart netfilter persistence if available
if systemctl list-unit-files 2>/dev/null | grep -q 'netfilter-persistent'; then
    systemctl restart netfilter-persistent 2>/dev/null || true
fi

echo "Done."
`

    cmd := exec.Command("bash", "-c", applyScript)
    cmd.Env = append(os.Environ(), "XSL_ENTRY="+entry, "TERM=xterm")
    out, err := cmd.CombinedOutput()
    if err != nil {
        logger.Error("Torrent RemoveTracker Apply Error:", string(out), err)
        ctx.JSON(http.StatusOK, gin.H{"success": false, "msg": "Entry removed from list but iptables cleanup failed: " + string(out)})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Tracker removed and unblocked successfully"})
}
