package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	uuid "github.com/satori/go.uuid"
)

// APIToken API Token 数据结构
type APIToken struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

var (
	apiTokens   map[string]APIToken
	apiTokensMu sync.RWMutex
)

// UserStatsEntry 用户统计数据结构
type UserStatsEntry struct {
	TotalCount     int            `json:"total_count"`
	StreakCount    int            `json:"streak_count"`
	CurrentStreak  int            `json:"current_streak"`
	LastMonthCount int            `json:"last_month_count"`
	LastFortuneDate string        `json:"last_fortune_date"`
	DailyCounts    map[string]int `json:"daily_counts"`
	ComputedAt     string         `json:"computed_at"`
}

var (
	userStatsCache map[int64]UserStatsEntry
	userStatsMu    sync.RWMutex
)

// loadAPITokens 从 DB 加载 API Tokens 到内存
func loadAPITokens() {
	apiTokens = make(map[string]APIToken)
	if err := db.Read("datas", "api_tokens", &apiTokens); err != nil {
		fmt.Println("API Token 数据不存在或加载失败，初始化为空")
		apiTokens = make(map[string]APIToken)
	}
	fmt.Printf("已加载 %d 个 API Token\n", len(apiTokens))
}

// saveAPITokens 持久化 API Tokens 到 DB
func saveAPITokens() {
	if err := db.Write("datas", "api_tokens", apiTokens); err != nil {
		fmt.Println("保存 API Token 失败:", err)
	}
}

// generateAPIToken 生成新 API Token
func generateAPIToken(name string) APIToken {
	raw := make([]byte, 32)
	rand.Read(raw)
	tokenStr := hex.EncodeToString(raw)

	token := APIToken{
		ID:        uuid.NewV4().String(),
		Token:     tokenStr,
		Name:      name,
		CreatedAt: time.Now().Format(gTimeFormat),
	}

	apiTokensMu.Lock()
	apiTokens[token.ID] = token
	saveAPITokens()
	apiTokensMu.Unlock()

	return token
}

// deleteAPIToken 删除指定 API Token
func deleteAPIToken(id string) bool {
	apiTokensMu.Lock()
	defer apiTokensMu.Unlock()
	if _, ok := apiTokens[id]; !ok {
		return false
	}
	delete(apiTokens, id)
	saveAPITokens()
	return true
}

// listAPITokens 返回所有 Token 列表（脱敏）
func listAPITokens() []APIToken {
	apiTokensMu.RLock()
	defer apiTokensMu.RUnlock()
	result := make([]APIToken, 0, len(apiTokens))
	for _, t := range apiTokens {
		display := t
		if len(display.Token) > 16 {
			display.Token = display.Token[:8] + "****" + display.Token[len(display.Token)-8:]
		}
		result = append(result, display)
	}
	return result
}

// validateAPIToken 验证 API Token 是否有效
func validateAPIToken(tokenStr string) bool {
	apiTokensMu.RLock()
	defer apiTokensMu.RUnlock()
	for _, t := range apiTokens {
		if t.Token == tokenStr {
			return true
		}
	}
	return false
}

// APITokenMiddleware Fiber 中间件：验证 API Token
func APITokenMiddleware(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "缺少 Authorization header"})
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if !validateAPIToken(tokenStr) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "API Token 无效"})
	}
	return c.Next()
}

// --- Token 管理 Handlers ---

// handleListTokens 列出所有 API Token（脱敏）
func handleListTokens(c fiber.Ctx) error {
	tokens := listAPITokens()
	return c.JSON(fiber.Map{"tokens": tokens})
}

// handleCreateToken 创建新 API Token
func handleCreateToken(c fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "请求格式错误"})
	}
	token := generateAPIToken(req.Name)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         token.ID,
		"token":      token.Token,
		"name":       token.Name,
		"created_at": token.CreatedAt,
	})
}

// handleDeleteToken 删除指定 API Token
func handleDeleteToken(c fiber.Ctx) error {
	id := c.Params("id")
	if !deleteAPIToken(id) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Token 不存在"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// --- 统计引擎 ---

// loadUserStats 从 DB 加载用户统计缓存到内存
func loadUserStats() {
	userStatsCache = make(map[int64]UserStatsEntry)
	if err := db.Read("datas", "user_stats", &userStatsCache); err != nil {
		fmt.Println("用户统计数据不存在或加载失败，初始化为空")
		userStatsCache = make(map[int64]UserStatsEntry)
	}
	fmt.Printf("已加载 %d 条用户统计\n", len(userStatsCache))
}

// saveUserStats 持久化用户统计缓存到 DB
func saveUserStats() {
	if err := db.Write("datas", "user_stats", userStatsCache); err != nil {
		fmt.Println("保存用户统计失败:", err)
	}
}

// computeUserStats 全量计算指定用户的统计信息
func computeUserStats(userID int64) UserStatsEntry {
	today := time.Now().Format("2006-01-02")
	entry := UserStatsEntry{
		DailyCounts: make(map[string]int),
		ComputedAt:  time.Now().Format(gTimeFormat),
	}

	// 从 streak 获取基础数据
	if s, ok := laoHL.userStreak[userID]; ok {
		entry.TotalCount = s.All
		entry.StreakCount = s.Weeks
		entry.CurrentStreak = s.Streak
		entry.LastFortuneDate = s.Date
	}

	// 扫描历史文件（最近 30 天）
	historyDir := "../db/history"
	entries, err := os.ReadDir(historyDir)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".json") {
				continue
			}
			dateStr := strings.TrimSuffix(name, ".json")
			fileDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			if time.Since(fileDate) > 30*24*time.Hour {
				continue
			}
			var cache laohuangliCache
			data, err := os.ReadFile(filepath.Join(historyDir, name))
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &cache); err != nil {
				continue
			}
			// history JSON 的 caches key 是 string 形式的 user ID
			if _, ok := cache.Caches[userID]; ok {
				entry.DailyCounts[dateStr]++
			}
		}
	}

	// 检查今日缓存
	if laoHL.cache.Date == today {
		if _, ok := laoHL.cache.Caches[userID]; ok {
			entry.DailyCounts[today]++
		}
	}

	// 计算 last_month_count
	entry.LastMonthCount = 0
	for _, count := range entry.DailyCounts {
		entry.LastMonthCount += count
	}

	return entry
}

// updateUserStatsOnFortune 算命成功后增量更新用户统计
func updateUserStatsOnFortune(userID int64) {
	today := time.Now().Format("2006-01-02")

	userStatsMu.Lock()
	defer userStatsMu.Unlock()

	entry, ok := userStatsCache[userID]
	if !ok {
		entry = UserStatsEntry{
			DailyCounts: make(map[string]int),
			ComputedAt:  time.Now().Format(gTimeFormat),
		}
	}

	// 从 streak 同步最新数据
	if s, ok := laoHL.userStreak[userID]; ok {
		entry.TotalCount = s.All
		entry.StreakCount = s.Weeks
		entry.CurrentStreak = s.Streak
		entry.LastFortuneDate = s.Date
	}

	entry.DailyCounts[today]++
	entry.LastMonthCount++
	entry.ComputedAt = time.Now().Format(gTimeFormat)

	userStatsCache[userID] = entry
	saveUserStats()
}

// expireUserStatsDaily 每日零点清理过期的每日统计
func expireUserStatsDaily() {
	userStatsMu.Lock()
	defer userStatsMu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	dirty := false
	for uid, entry := range userStatsCache {
		total := 0
		for date, count := range entry.DailyCounts {
			if date < cutoff {
				delete(entry.DailyCounts, date)
				dirty = true
			} else {
				total += count
			}
		}
		entry.LastMonthCount = total
		entry.ComputedAt = time.Now().Format(gTimeFormat)
		userStatsCache[uid] = entry
	}
	if dirty {
		saveUserStats()
	}
}

// handleUserStats 用户统计 API handler
func handleUserStats(c fiber.Ctx) error {
	idStr := c.Params("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "无效的用户 ID"})
	}

	// 检查用户是否存在
	if _, ok := laoHL.userStreak[userID]; !ok {
		// 检查历史记录中是否有该用户
		found := false
		historyDir := "../db/history"
		if dirEntries, err := os.ReadDir(historyDir); err == nil {
			for _, e := range dirEntries {
				name := e.Name()
				if !strings.HasSuffix(name, ".json") {
					continue
				}
				fileDate, _ := time.Parse("2006-01-02", strings.TrimSuffix(name, ".json"))
				if time.Since(fileDate) > 30*24*time.Hour {
					continue
				}
				data, err := os.ReadFile(filepath.Join(historyDir, name))
				if err != nil {
					continue
				}
				var cache laohuangliCache
				if err := json.Unmarshal(data, &cache); err != nil {
					continue
				}
				if _, ok := cache.Caches[userID]; ok {
					found = true
					break
				}
			}
		}
		if !found {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "用户不存在或从未算命"})
		}
	}

	// 优先使用缓存，缓存不存在则全量计算
	userStatsMu.RLock()
	entry, ok := userStatsCache[userID]
	userStatsMu.RUnlock()

	if !ok {
		entry = computeUserStats(userID)
		userStatsMu.Lock()
		userStatsCache[userID] = entry
		saveUserStats()
		userStatsMu.Unlock()
	}

	return c.JSON(fiber.Map{
		"user_id":           idStr,
		"total_count":       entry.TotalCount,
		"streak_count":      entry.StreakCount,
		"current_streak":    entry.CurrentStreak,
		"last_month_count":  entry.LastMonthCount,
		"last_fortune_date": entry.LastFortuneDate,
	})
}
