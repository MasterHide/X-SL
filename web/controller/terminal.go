package controller

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"x-ui/logger"
	"x-ui/web/service"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TerminalSession struct {
	Token     string
	ExpiresAt time.Time
	Pty       *os.File
	Mutex     sync.Mutex
}

type TerminalController struct {
	settingService *service.SettingService
	sessions       map[string]*TerminalSession
	mu             sync.Mutex
}

func NewTerminalController(s *service.SettingService) *TerminalController {
	return &TerminalController{
		settingService: s,
		sessions:       make(map[string]*TerminalSession),
	}
}

// Index renders the terminal HTML page
func (c *TerminalController) Index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "terminal.html", gin.H{})
}

// SetPassword is used during the setup wizard to set the terminal password
func (c *TerminalController) SetPassword(ctx *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.Password) < 8 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
		return
	}

	// 1. Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// 2. Save the hash to the database via SettingService
	err = c.settingService.SetTerminalPasswordHash(string(hashedPassword))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save password"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// Authenticate checks the password and issues a 10-minute session token
func (c *TerminalController) Authenticate(ctx *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 1. Fetch hashed password from settings database
	hash, err := c.settingService.GetTerminalPasswordHash()
	if err != nil || hash == "" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Terminal access is not configured."})
		return
	}

	// 2. Compare password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// 3. Generate secure temporary token
	token := make([]byte, 32)
	rand.Read(token)
	tokenStr := hex.EncodeToString(token)

	// 4. Store session with 10-minute expiration
	c.mu.Lock()
	c.sessions[tokenStr] = &TerminalSession{
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	c.mu.Unlock()

	ctx.JSON(http.StatusOK, gin.H{
		"token":   tokenStr,
		"expires": 600,
	})
}

// HandleWebSocket establishes the terminal connection
func (c *TerminalController) HandleWebSocket(ctx *gin.Context) {
	token := ctx.Query("token")

	// 1. Validate token
	c.mu.Lock()
	session, exists := c.sessions[token]
	c.mu.Unlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return
	}

	// 2. Upgrade to WebSocket
	ws, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed:", err)
		return
	}
	defer ws.Close()

	// 3. Start PTY shell
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe")
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		cmd = exec.Command(shell)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		logger.Error("Failed to start PTY:", err)
		ws.Close()
		return
	}
	defer ptmx.Close()

	// 4. Bridge WebSocket and PTY
	done := make(chan struct{})

	// PTY -> WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			err = ws.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				break
			}
		}
	}()

	// WebSocket -> PTY
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}

			var resize struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if err := json.Unmarshal(msg, &resize); err == nil && resize.Type == "resize" {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(resize.Cols), Rows: uint16(resize.Rows)})
				continue
			}

			ptmx.Write(msg)
		}
	}()

	// 5. Session Timer (10 minutes)
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()

	select {
	case <-done:
		// Connection closed naturally
	case <-timer.C:
		// Force disconnect after 10 minutes
		ws.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mSession expired. Disconnecting...\x1b[0m\r\n"))
		ws.Close()
		ptmx.Close()
	}

	// Cleanup session
	c.mu.Lock()
	delete(c.sessions, token)
	c.mu.Unlock()
}
