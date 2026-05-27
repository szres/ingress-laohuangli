package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	openai "github.com/sashabaranov/go-openai"
)

type laohuangliResult struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

type laohuangliCache struct {
	Date   string                     `json:"date"`
	Caches map[int64]laohuangliResult `json:"caches"`
}

type AnnualSummary struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type AnnualUserStats struct {
	ID           int64
	Name         string
	AICount      int
	ActiveDays   int
	FirstDate    string
	LastDate     string
	Quotes       []string
	RankPercent  string
	SummaryQuote string
}

type SummaryRequest struct {
	Name        string   `json:"name"`
	TotalDays   int      `json:"total_days"`
	ActiveDays  int      `json:"active_days"`
	AICount     int      `json:"ai_count"`
	FirstDate   string   `json:"first_date"`
	LastDate    string   `json:"last_date"`
	RankPercent string   `json:"rank_percent"`
	Candidates  []string `json:"candidate_quotes"`
}

type LLMConfig struct {
	Model string
}

type LLMClient struct {
	client *openai.Client
	cfg    LLMConfig
}

type QuoteCandidate struct {
	Text  string
	Score int
}

func main() {
	year := flag.Int("year", 0, "统计年份（例如 2024）")
	dbPath := flag.String("db", "../db", "db 目录路径")
	outPath := flag.String("out", "", "输出 annual json 文件路径")
	threshold := flag.Int("threshold", 30, "进入 LLM 总结的最小次数")
	maxCandidates := flag.Int("candidates", 30, "每人候选语句数量上限")
	fallbackPath := flag.String("fallback", "", "fallback 文案 json 路径（用于读取或写入）")
	generateFallback := flag.Bool("generate-fallback", false, "生成 fallback 文案并写入 fallback 路径")
	dryRun := flag.Bool("dry-run", false, "仅打印统计信息，不写文件")
	flag.Parse()

	if *year <= 0 {
		exitWithError(errors.New("必须指定 -year"))
	}
	if *outPath == "" {
		*outPath = filepath.Join(*dbPath, "annual", fmt.Sprintf("%d.json", *year))
	}

	llmClient, err := newLLMClient()
	if err != nil {
		exitWithError(err)
	}

	if *generateFallback {
		if *fallbackPath == "" {
			*fallbackPath = filepath.Join(*dbPath, "annual", "fallback.json")
		}
		existingFallbacks, _ := loadFallbacks("")
		if err := generateFallbacks(llmClient, *fallbackPath, existingFallbacks); err != nil {
			exitWithError(err)
		}
		fmt.Printf("fallback 文案已生成: %s\n", *fallbackPath)
		return
	}

	annual, stats, err := buildAnnualSummary(*dbPath, *outPath, *year, *threshold, *maxCandidates, llmClient, *fallbackPath)
	if err != nil {
		exitWithError(err)
	}

	fmt.Printf("统计完成：用户数 %d\n", len(stats))
	if *dryRun {
		return
	}

	if err := writeJSON(*outPath, annual); err != nil {
		exitWithError(fmt.Errorf("写入输出失败: %w", err))
	}
	fmt.Printf("年度总结已写入: %s\n", *outPath)
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
	os.Exit(1)
}

func newLLMClient() (*LLMClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY 未设置")
	}
	config := openai.DefaultConfig(apiKey)
	if os.Getenv("OPENAI_BASE_URL") != "" {
		config.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = openai.GPT4oMini
	}
	return &LLMClient{
		client: openai.NewClientWithConfig(config),
		cfg: LLMConfig{
			Model: model,
		},
	}, nil
}

func buildAnnualSummary(dbPath string, outPath string, year int, threshold int, maxCandidates int, llm *LLMClient, fallbackPath string) (map[string]AnnualSummary, []AnnualUserStats, error) {
	historyDir := filepath.Join(dbPath, "history")

	annual := make(map[string]AnnualSummary)
	if existing, err := loadAnnualFile(outPath); err == nil {
		annual = existing
	}

	statsMap := make(map[int64]*AnnualUserStats)
	allEntries := readHistoryEntriesForYear(historyDir, year)
	yearlyDates := make([]string, 0, len(allEntries))
	for dateStr, cache := range allEntries {
		yearlyDates = append(yearlyDates, dateStr)
		for id, result := range cache.Caches {
			stat := getOrInitStats(statsMap, id, result.Name)
			stat.ActiveDays++
			aiCount := countAI(result.Result)
			stat.AICount += aiCount
			updateDateRange(stat, dateStr)
			updateQuotes(stat, result.Result)
		}
	}

	stats := make([]AnnualUserStats, 0, len(statsMap))
	for _, stat := range statsMap {
		stat.Quotes = pickTopQuotes(stat.Quotes, maxCandidates)
		stats = append(stats, *stat)
	}
	applyRank(stats, threshold)
	activeDaysTotal := len(yearlyDates)

	fallbackQuotes, err := loadFallbacks(fallbackPath)
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].ActiveDays == stats[j].ActiveDays {
			return stats[i].ID < stats[j].ID
		}
		return stats[i].ActiveDays > stats[j].ActiveDays
	})
	eligibleCount := 0
	for _, stat := range stats {
		if stat.ActiveDays > threshold {
			eligibleCount++
		}
	}
	processedEligible := 0
	for _, stat := range stats {
		key := strconv.FormatInt(stat.ID, 10)
		if stat.ActiveDays > threshold {
			if _, exists := annual[key]; exists {
				processedEligible++
				fmt.Printf("Progress: %d/%d/%d\n", processedEligible, eligibleCount, len(stats))
				continue
			}
		}
		content := ""
		if stat.ActiveDays > threshold {
			processedEligible++
			fmt.Printf("Progress: %d/%d/%d\n", processedEligible, eligibleCount, len(stats))
			summary, err := llm.GenerateSummary(buildSummaryRequest(stat, activeDaysTotal), stat.Quotes)
			if err != nil {
				content = randomFallback(fallbackQuotes)
			} else {
				content = summary
			}
		} else {
			content = randomFallback(fallbackQuotes)
		}
		annual[key] = AnnualSummary{
			Name:    stat.Name,
			Content: content,
		}
		if err := writeJSON(outPath, annual); err != nil {
			return nil, nil, fmt.Errorf("写入断点文件失败: %w", err)
		}
	}
	if len(yearlyDates) == 0 {
		fmt.Println("⚠️ 未找到指定年份的历史记录")
	}
	return annual, stats, nil
}

func getOrInitStats(stats map[int64]*AnnualUserStats, id int64, name string) *AnnualUserStats {
	if _, ok := stats[id]; !ok {
		stats[id] = &AnnualUserStats{ID: id, Name: name, Quotes: make([]string, 0)}
	}
	return stats[id]
}

func updateDateRange(stat *AnnualUserStats, dateStr string) {
	if stat.FirstDate == "" || dateStr < stat.FirstDate {
		stat.FirstDate = dateStr
	}
	if stat.LastDate == "" || dateStr > stat.LastDate {
		stat.LastDate = dateStr
	}
}

func countAI(result string) int {
	re := regexp.MustCompile(`（AI:(\d+)）`)
	match := re.FindStringSubmatch(result)
	if len(match) > 1 {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return n
		}
	}
	return 0
}

func updateQuotes(stat *AnnualUserStats, result string) {
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		return
	}
	main := lines[len(lines)-1]
	parts := strings.Split(main, "，")
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.TrimSuffix(part, "。"))
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "宜") || strings.HasPrefix(trimmed, "忌") {
			stat.Quotes = append(stat.Quotes, trimmed)
		}
	}
}

func pickTopQuotes(quotes []string, limit int) []string {
	scored := make([]QuoteCandidate, 0, len(quotes))
	for _, q := range quotes {
		score := quoteScore(q)
		scored = append(scored, QuoteCandidate{Text: q, Score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Text > scored[j].Text
		}
		return scored[i].Score > scored[j].Score
	})
	unique := make([]string, 0, len(scored))
	seen := make(map[string]struct{})
	for _, c := range scored {
		key := normalizeQuote(c.Text)
		if _, ok := seen[key]; ok {
			continue
		}
		unique = append(unique, c.Text)
		seen[key] = struct{}{}
		if len(unique) >= limit {
			break
		}
	}
	if len(unique) == 0 {
		return unique
	}
	rand.Shuffle(len(unique), func(i, j int) {
		unique[i], unique[j] = unique[j], unique[i]
	})
	return unique
}

func applyRank(stats []AnnualUserStats, threshold int) {
	if len(stats) == 0 {
		return
	}
	sorted := make([]AnnualUserStats, 0, len(stats))
	for _, stat := range stats {
		if stat.ActiveDays > threshold {
			sorted = append(sorted, stat)
		}
	}
	if len(sorted) == 0 {
		for i := range stats {
			stats[i].RankPercent = ""
		}
		return
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ActiveDays > sorted[j].ActiveDays
	})
	total := float64(len(sorted))
	rankMap := make(map[int64]string, len(sorted))
	for idx, stat := range sorted {
		percentile := float64(idx+1) / total
		percent := int(math.Ceil(percentile * 100))
		if percent < 1 {
			percent = 1
		}
		rankMap[stat.ID] = fmt.Sprintf("前%d%%", percent)
	}
	for i := range stats {
		stats[i].RankPercent = rankMap[stats[i].ID]
	}
}

func buildSummaryRequest(stat AnnualUserStats, totalDays int) SummaryRequest {
	return SummaryRequest{
		Name:        stat.Name,
		TotalDays:   totalDays,
		ActiveDays:  stat.ActiveDays,
		AICount:     stat.AICount,
		FirstDate:   stat.FirstDate,
		LastDate:    stat.LastDate,
		RankPercent: stat.RankPercent,
		Candidates:  stat.Quotes,
	}
}

func (llm *LLMClient) GenerateSummary(req SummaryRequest, candidates []string) (string, error) {
	payload, _ := json.MarshalIndent(req, "", "  ")
	prompt := "你是Ingress老黄历年终总结的写手，请根据以下用户数据，写一段100-200字的中文年终总结。风格幽默、有点损但友好，允许引用候选语句中的梗。请至少引用或轻微改写1条候选语句，候选语句涉及Ingress相关名词请保持英文。不要输出列表或JSON，只输出总结正文。\n\n字段说明：\n- TotalDays：统计的总天数。\n- ActiveDays：统计期间用户进行了算命的天数。\n- AICount：AI提供的算命结果的总条数（每次算命由AI提供的结果可能为0-2条）。\n- FirstDate：统计期间第一次算命的日期。\n- LastDate：统计期间最后一次算命的日期。\n\n用户数据:\n" + string(payload)
	if len(candidates) == 0 {
		prompt += "\n候选语句为空，请自己发挥但仍保持Ingress玩家语境。"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	resp, err := llm.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llm.cfg.Model,
		Messages: []openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		}},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func generateFallbacks(llm *LLMClient, outputPath string, samples []string) error {
	sampleText := ""
	if len(samples) > 0 {
		maxSample := min(6, len(samples))
		for i := 0; i < maxSample; i++ {
			sampleText += fmt.Sprintf("%d) %s\n", i+1, samples[i])
		}
	}
	prompt := "你是Ingress老黄历年终总结的写手，请生成40条面向算命次数不足用户的短句，每条40-80字。语气幽默、略带调侃但友好，尽量使用Ingress玩家语境。不要编号，不要空行，每行一条。"
	if sampleText != "" {
		prompt += "\n以下是参考示例，请保持风格一致：\n" + sampleText
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resp, err := llm.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llm.cfg.Model,
		Messages: []openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		}},
	})
	if err != nil {
		return err
	}
	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	fallbacks := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fallbacks = append(fallbacks, trimmed)
	}
	if len(fallbacks) == 0 {
		return errors.New("LLM 未返回 fallback 文案")
	}
	return writeJSON(outputPath, fallbacks)
}

func loadFallbacks(path string) ([]string, error) {
	if path == "" {
		return []string{
			"今年你的算命次数不够，像是只在Portal边路过了一次，期待你明年多来刷几发XMP。",
			"算命次数偏少，像是只做了两次Glyph Hack就跑路，Ingress宇宙还等你补票。",
			"你对命运的关注度有点低，建议明年多点几次，别让Resonator白搭。",
		}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 fallback 失败: %w", err)
	}
	var fallbacks []string
	if err := json.Unmarshal(data, &fallbacks); err != nil {
		return nil, fmt.Errorf("解析 fallback 失败: %w", err)
	}
	if len(fallbacks) == 0 {
		return nil, errors.New("fallback 文案为空")
	}
	return fallbacks, nil
}

func randomFallback(fallbacks []string) string {
	if len(fallbacks) == 0 {
		return "今年你的算命次数不够，明年再来刷爆命运池。"
	}
	return fallbacks[rand.IntN(len(fallbacks))]
}

type monthlyArchive map[string]laohuangliCache

func readHistoryEntriesForYear(historyDir string, year int) map[string]laohuangliCache {
	result := make(map[string]laohuangliCache)

	dirEntries, err := os.ReadDir(historyDir)
	if err != nil {
		return result
	}

	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := e.Name()
		base := strings.TrimSuffix(name, ".json")

		if _, err := time.Parse("2006-01", base); err == nil {
			var archive monthlyArchive
			data, err := os.ReadFile(filepath.Join(historyDir, name))
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &archive); err != nil {
				continue
			}
			for dateStr, cache := range archive {
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					continue
				}
				if t.Year() == year {
					result[dateStr] = cache
				}
			}
			continue
		}

		fileDate, err := time.Parse("2006-01-02", base)
		if err != nil {
			continue
		}
		if fileDate.Year() != year {
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
		result[base] = cache
	}

	return result
}

func writeJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o644)
}

func loadAnnualFile(path string) (map[string]AnnualSummary, error) {
	if path == "" {
		return nil, errors.New("annual 输出路径为空")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var annual map[string]AnnualSummary
	if err := json.Unmarshal(data, &annual); err != nil {
		return nil, err
	}
	if annual == nil {
		return nil, errors.New("annual 为空")
	}
	return annual, nil
}

func quoteScore(q string) int {
	trimmed := strings.TrimSpace(q)
	length := utf8.RuneCountInString(trimmed)
	if length == 0 {
		return 0
	}
	score := length
	if length >= 10 && length <= 26 {
		score += 8
	}
	if strings.Contains(trimmed, "但是") || strings.Contains(trimmed, "却") || strings.Contains(trimmed, "不过") {
		score += 3
	}
	if strings.ContainsAny(trimmed, "!?？！") {
		score += 2
	}
	if strings.Count(trimmed, "，") >= 2 {
		score += 3
	}
	if strings.Contains(trimmed, "不要") || strings.Contains(trimmed, "一定") {
		score += 2
	}
	return score
}

func normalizeQuote(q string) string {
	trimmed := strings.TrimSpace(q)
	trimmed = strings.TrimSuffix(trimmed, "。")
	trimmed = strings.TrimSuffix(trimmed, "！")
	trimmed = strings.TrimSuffix(trimmed, "!")
	trimmed = strings.TrimSuffix(trimmed, "？")
	trimmed = strings.TrimSuffix(trimmed, "?")
	return strings.ToLower(trimmed)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
