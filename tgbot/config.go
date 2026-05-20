package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// normalizeDomain 剥离域名中的协议前缀和尾部斜杠
// "https://example.com/" → "example.com"
func normalizeDomain(raw string) string {
	domain := strings.TrimSpace(raw)
	if domain == "" {
		return ""
	}
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimRight(domain, "/")
	return domain
}

// AppConfig 应用配置，存储在 scribble DB 中
type AppConfig struct {
	BotToken      string `json:"bot_token"`
	AdminID       string `json:"admin_id"`
	KumaPushURL   string `json:"kuma_push_url"`
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"`
	OpenAIModel   string `json:"openai_model"`
	AdminUsername string `json:"admin_username"`
	AdminPassword string `json:"admin_password"` // bcrypt 哈希
	WebDomain     string `json:"web_domain"`     // Webhook + 前端域名
}

var appConfig AppConfig
var configMu sync.RWMutex

// ConfigJSON 用于 API 返回时脱敏
type ConfigJSON struct {
	BotToken      string `json:"bot_token"`
	AdminID       string `json:"admin_id"`
	KumaPushURL   string `json:"kuma_push_url"`
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"`
	OpenAIModel   string `json:"openai_model"`
	AdminUsername string `json:"admin_username"`
	WebDomain     string `json:"web_domain"`
}

// maskString 对敏感字符串脱敏，保留前4位和后4位
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// ToJSON 返回脱敏后的配置
func (c *AppConfig) ToJSON() ConfigJSON {
	return ConfigJSON{
		BotToken:      maskString(c.BotToken),
		AdminID:       c.AdminID,
		KumaPushURL:   c.KumaPushURL,
		OpenAIAPIKey:  maskString(c.OpenAIAPIKey),
		OpenAIBaseURL: c.OpenAIBaseURL,
		OpenAIModel:   c.OpenAIModel,
		AdminUsername: c.AdminUsername,
		WebDomain:     c.WebDomain,
	}
}

// loadConfig 从 DB 加载配置，如果不存在则从环境变量初始化
func loadConfig() {
	err := db.Read("datas", "config", &appConfig)
	if err != nil || appConfig.AdminUsername == "" {
		fmt.Println("配置不存在或不完整，从环境变量初始化...")
		initConfigFromEnv()
		return
	}
	fmt.Println("从数据库加载配置成功")
}

// initConfigFromEnv 初始化默认配置并写入 DB
// 管理员账户从环境变量读取（首次登录必需），其余配置通过 Web 管理面板设置
func initConfigFromEnv() {
	configMu.Lock()
	defer configMu.Unlock()

	// Bot Token、Admin ID、OpenAI 配置等均通过 Web 管理面板设置
	appConfig.BotToken = ""
	appConfig.AdminID = ""
	appConfig.KumaPushURL = os.Getenv("KUMA_PUSH_URL")
	appConfig.OpenAIAPIKey = ""
	appConfig.OpenAIBaseURL = ""
	appConfig.OpenAIModel = ""
	appConfig.WebDomain = normalizeDomain(os.Getenv("WEB_DOMAIN"))

	// 管理员账户：从环境变量读取，或使用默认值
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "laohuangli"
	}
	appConfig.AdminUsername = adminUser

	// 对密码进行 bcrypt 哈希
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码哈希失败:", err)
		return
	}
	appConfig.AdminPassword = string(hashedPass)

	if err := db.Write("datas", "config", appConfig); err != nil {
		fmt.Println("保存初始配置失败:", err)
	} else {
		fmt.Println("初始配置已保存到数据库")
	}
}

// GetBotToken 获取 Bot Token
func GetBotToken() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.BotToken
}

// GetAdminID 获取管理员 Telegram ID
func GetAdminID() int64 {
	configMu.RLock()
	defer configMu.RUnlock()
	id, _ := strconv.ParseInt(appConfig.AdminID, 10, 64)
	return id
}

// GetKumaPushURL 获取 Kuma Push URL
func GetKumaPushURL() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.KumaPushURL
}

// GetOpenAIAPIKey 获取 OpenAI API Key
func GetOpenAIAPIKey() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.OpenAIAPIKey
}

// GetOpenAIBaseURL 获取 OpenAI Base URL
func GetOpenAIBaseURL() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.OpenAIBaseURL
}

// GetOpenAIModel 获取 OpenAI 模型名称
func GetOpenAIModel() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.OpenAIModel
}

// UpdateConfig 更新配置（部分更新）
func UpdateConfig(update AppConfig) {
	configMu.Lock()
	defer configMu.Unlock()

	// 只更新非空字段
	if update.BotToken != "" {
		appConfig.BotToken = update.BotToken
	}
	if update.AdminID != "" {
		appConfig.AdminID = update.AdminID
	}
	if update.KumaPushURL != "" {
		appConfig.KumaPushURL = update.KumaPushURL
	}
	if update.OpenAIAPIKey != "" {
		appConfig.OpenAIAPIKey = update.OpenAIAPIKey
	}
	if update.OpenAIBaseURL != "" {
		appConfig.OpenAIBaseURL = update.OpenAIBaseURL
	}
	if update.OpenAIModel != "" {
		appConfig.OpenAIModel = update.OpenAIModel
	}
	if update.AdminUsername != "" {
		appConfig.AdminUsername = update.AdminUsername
	}
	if update.AdminPassword != "" {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(update.AdminPassword), bcrypt.DefaultCost)
		if err == nil {
			appConfig.AdminPassword = string(hashedPass)
		}
	}
	if update.WebDomain != "" {
		appConfig.WebDomain = normalizeDomain(update.WebDomain)
	}

	db.Write("datas", "config", appConfig)
}

// VerifyPassword 验证密码
func VerifyPassword(password string) bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return bcrypt.CompareHashAndPassword([]byte(appConfig.AdminPassword), []byte(password)) == nil
}

// GetUsername 获取管理员用户名
func GetUsername() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig.AdminUsername
}

// GetWebDomain 获取 Webhook + 前端域名（自动规范化）
func GetWebDomain() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return normalizeDomain(appConfig.WebDomain)
}
