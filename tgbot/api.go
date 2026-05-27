package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("laohuangli-jwt-secret-change-in-production")

// Claims JWT 声明
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWT 生成 JWT token
func GenerateJWT(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT 验证 JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// AuthMiddleware Fiber JWT 认证中间件
func AuthMiddleware(c fiber.Ctx) error {
	tokenStr := ""
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if tokenStr == "" {
		tokenStr = c.Cookies("auth_token")
	}
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "未授权"})
	}

	claims, err := ValidateJWT(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token 无效或已过期"})
	}

	c.Locals("username", claims.Username)
	return c.Next()
}

// handleLogin 处理登录请求
func handleLogin(c fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "请求格式错误"})
	}

	if req.Username != GetUsername() || !VerifyPassword(req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "用户名或密码错误"})
	}

	token, err := GenerateJWT(req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "生成 token 失败"})
	}

	return c.JSON(fiber.Map{"token": token})
}

// handleGetConfig 获取配置（脱敏）
func handleGetConfig(c fiber.Ctx) error {
	configMu.RLock()
	cfg := appConfig.ToJSON()
	configMu.RUnlock()
	return c.JSON(cfg)
}

// handleUpdateConfig 更新配置
func handleUpdateConfig(c fiber.Ctx) error {
	var update AppConfig
	if err := c.Bind().JSON(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "请求格式错误"})
	}

	botChanged := update.BotToken != "" || update.AdminID != "" || update.WebDomain != ""
	aiChanged := update.OpenAIAPIKey != "" || update.OpenAIBaseURL != "" || update.OpenAIModel != ""

	UpdateConfig(update)

	if botChanged {
		go func() {
			if restartBot() {
				fmt.Println("Telegram Bot 热重载成功")
			} else {
				fmt.Println("Telegram Bot 热重载失败")
			}
		}()
	}

	if aiChanged {
		go reloadAIConfig()
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// handleGetLogs 获取日志
func handleGetLogs(c fiber.Ctx) error {
	n := 200
	if nStr := c.Query("n"); nStr != "" {
		if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
			n = v
		}
	}

	lines := []string{}
	if logBuffer != nil {
		lines = logBuffer.GetLines(n)
	}

	return c.JSON(fiber.Map{"lines": lines})
}

// BrowserOnlyMiddleware Sec-Fetch-Site 浏览器校验（Forbidden Header，JS 无法伪造）
func BrowserOnlyMiddleware(c fiber.Ctx) error {
	secFetchSite := c.Get("Sec-Fetch-Site")
	if secFetchSite != "same-origin" && secFetchSite != "same-site" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "仅允许浏览器访问",
		})
	}
	return c.Next()
}

// handleTodayCount 获取今日已算命用户数量（公开 API）
func handleTodayCount(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"date":  laoHL.cache.Date,
		"count": len(laoHL.cache.Caches),
	})
}

// handleGetCache 获取今日缓存数据（今日指引 + 众生列表）
func handleGetCache(c fiber.Ctx) error {
	return c.JSON(laoHL.cache)
}

// handleGetTemplates 获取模板列表
func handleGetTemplates(c fiber.Ctx) error {
	return c.JSON(laoHL.templates)
}

// handleGetEntries 获取词条列表（本地 + 用户提名）
func handleGetEntries(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"entries":      laoHL.entries,
		"entries_user": laoHL.entriesUser,
	})
}

// SetupRoutes 注册所有 Fiber 路由
func SetupRoutes(app *fiber.App) {
	// 公开 API（速率限制：每 IP 每分钟最多 5 次）
	publicLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
	})
	app.Get("/api/today", BrowserOnlyMiddleware, publicLimiter, handleTodayCount)
	app.Get("/api/cache", BrowserOnlyMiddleware, publicLimiter, handleGetCache)
	app.Get("/api/templates", BrowserOnlyMiddleware, publicLimiter, handleGetTemplates)
	app.Get("/api/entries", BrowserOnlyMiddleware, publicLimiter, handleGetEntries)

	// 认证 API
	app.Post("/api/auth/login", handleLogin)

	// 管理 API（需要认证）
	app.Get("/api/admin/config", AuthMiddleware, handleGetConfig)
	app.Put("/api/admin/config", AuthMiddleware, handleUpdateConfig)
	app.Get("/api/admin/logs", AuthMiddleware, handleGetLogs)
	app.Get("/api/admin/tokens", AuthMiddleware, handleListTokens)
	app.Post("/api/admin/tokens", AuthMiddleware, handleCreateToken)
	app.Delete("/api/admin/tokens/:id", AuthMiddleware, handleDeleteToken)

	// 用户统计 API（需要 API Token 认证）
	app.Get("/api/user/:id/stats", APITokenMiddleware, handleUserStats)

	// Webhook 端点（Telegram 推送），路径中附带 bot token 防止冲突
	app.Post("/webhook/:token", handleWebhook)

	// 兼容旧的静态文件访问
	app.Get("/static/*", static.New("../db/datas", static.Config{
		Browse: false,
	}))
}
