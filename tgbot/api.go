package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// AuthMiddleware 认证中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Authorization header 或 cookie 获取 token
		tokenStr := ""
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if tokenStr == "" {
			cookie, err := r.Cookie("auth_token")
			if err == nil {
				tokenStr = cookie.Value
			}
		}
		if tokenStr == "" {
			http.Error(w, `{"error":"未授权"}`, http.StatusUnauthorized)
			return
		}

		claims, err := ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"token 无效或已过期"}`, http.StatusUnauthorized)
			return
		}

		// 将 username 存入 request context
		r.Header.Set("X-Username", claims.Username)
		next(w, r)
	}
}

// handleLogin 处理登录请求
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"请求格式错误"}`, http.StatusBadRequest)
		return
	}

	if req.Username != GetUsername() || !VerifyPassword(req.Password) {
		http.Error(w, `{"error":"用户名或密码错误"}`, http.StatusUnauthorized)
		return
	}

	token, err := GenerateJWT(req.Username)
	if err != nil {
		http.Error(w, `{"error":"生成 token 失败"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// handleGetConfig 获取配置（脱敏）
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	configMu.RLock()
	cfg := appConfig.ToJSON()
	configMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// handleUpdateConfig 更新配置
func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var update AppConfig
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error":"请求格式错误"}`, http.StatusBadRequest)
		return
	}

	// 检查是否有 bot 相关配置变更
	botChanged := update.BotToken != "" || update.AdminID != ""
	// 检查是否有 AI 配置变更
	aiChanged := update.OpenAIAPIKey != "" || update.OpenAIBaseURL != "" || update.OpenAIModel != ""

	UpdateConfig(update)

	// 如果 bot 配置变更，热重载 bot
	if botChanged {
		go func() {
			if restartBot() {
				fmt.Println("Telegram Bot 热重载成功")
			} else {
				fmt.Println("Telegram Bot 热重载失败")
			}
		}()
	}

	// 如果 AI 配置变更，热重载
	if aiChanged {
		go reloadAIConfig()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetLogs 获取日志
func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	n := 200 // 默认返回最近200行
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := strconv.Atoi(nStr); err == nil && v > 0 {
			n = v
		}
	}

	lines := []string{}
	if logBuffer != nil {
		lines = logBuffer.GetLines(n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"lines": lines})
}

// handleGetCache 获取今日缓存数据（公开 API）
func handleGetCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	result := map[string]interface{}{
		"date":   laoHL.cache.Date,
		"today":  laoHL.cache.Today,
		"caches": laoHL.cache.Caches,
	}
	json.NewEncoder(w).Encode(result)
}

// handleGetTemplates 获取模板列表（公开 API）
func handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(laoHL.templates)
}

// handleGetEntries 获取词条列表（公开 API，合并系统+用户词条）
func handleGetEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	result := map[string]interface{}{
		"entries":      laoHL.entries,
		"entries_user": laoHL.entriesUser,
	}
	json.NewEncoder(w).Encode(result)
}

// StartAPIServer 启动 API 服务器，替代原来的 StartFileServer
func StartAPIServer() {
	mux := http.NewServeMux()

	// 公开 API
	mux.HandleFunc("/api/cache", handleGetCache)
	mux.HandleFunc("/api/templates", handleGetTemplates)
	mux.HandleFunc("/api/entries", handleGetEntries)

	// 认证 API
	mux.HandleFunc("/api/auth/login", handleLogin)

	// 管理 API（需要认证）
	mux.HandleFunc("/api/admin/config", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetConfig(w, r)
		case http.MethodPut:
			handleUpdateConfig(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/admin/logs", AuthMiddleware(handleGetLogs))

	// 保持兼容：旧的静态文件服务（JSON 文件直接访问）
	mux.Handle("/", http.FileServer(http.Dir("../db/datas")))

	fmt.Println("API 服务器启动，监听 :80")
	err := http.ListenAndServe(":80", mux)
	if err != nil {
		panic(err)
	}
}
