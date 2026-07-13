package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

var openaiClient *openai.Client
var AISamples []string = []string{
	"拒接领导电话",
	"翘班去钓鱼",
	"边砍圣诞树边刷AP",
	"投食减肥者",
	"变得不幸",
	"橙色针织裙",
	"搭星舰去上班",
	"带猫猫参加IFS",
	"胡萝卜玉米猪骨汤",
}

var defaultGoodSamples = []string{
	"拒接领导电话", "翘班去钓鱼", "边砍圣诞树边刷AP", "投食减肥者", "变得不幸",
	"橙色针织裙", "搭星舰去上班", "带猫猫参加IFS", "胡萝卜玉米猪骨汤", "给Portal贴膜",
	"把XMP当烟花", "用ADA处理前任", "在Link上走钢丝", "给Scanner充电到忘记睡觉",
	"把低电量当人生哲学", "穿拖鞋参加战术会议", "骑共享单车追稀有Portal",
	"在咖啡里找XM", "把通勤路线画成Field", "和敌对阵营拼桌", "给背包做减法",
	"在雨里更新Scanner", "把钥匙串当护身符", "给B8许愿", "假装看不见群消息",
	"把午休献给Portal", "在地铁里规划大三角", "为一根Link熬夜", "把IFS当相亲局",
	"用Jarvis解决选择困难", "被风吹乱战术发型", "给外卖备注阵营色", "在奶茶里加抵抗",
	"把加班解释为刷AP", "对着地图假装很忙", "把雨伞当Portal天线", "给鞋带打战术结",
	"把迷路称为实地勘测", "在凌晨维护社交能量", "给闹钟设置成Scanner提示音",
}

var promptDefault = "你是 Ingress 主题的老黄历词条生成器。词条范围包含与日期和小时相关的活动、Ingress 游戏行为、穿搭、交通、饮食、网络迷因和流行梗；Ingress 专有名词必须使用英文。词条应简短、搞笑且有讽刺感，不含“宜”或“忌”前缀、逗号或大括号，且不超过 64 个 Unicode 字符。每条必须同时能自然接在“宜”和“忌”之后。不要重复示例或同一批中的其他词条。\n"

func GetChineseWeekday(t time.Time) string {
	// 数组下标 0-6 分别对应周日到周六
	weekdays := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	return weekdays[t.Weekday()]
}
func todayChineseDateTime(t time.Time) string {
	return fmt.Sprintf("%d年%d月%d日%s%d点", t.Year(), t.Month(), t.Day(), GetChineseWeekday(t), t.Hour())
}

type AIInstance struct {
	Name   string
	Init   func(*AIInstance)
	Update func(*AIInstance, time.Time, *[]string) error
	Valid  bool
}

var AIs []*AIInstance

var aiContext, cancelAI = context.WithCancel(context.Background())
var aiGenerationMu sync.Mutex

// 每个模型独立退避；配置顺序同时是调用优先级。
type openaiModelState struct {
	nextRetry    time.Time
	failureDelay time.Duration
}

var (
	openaiModelMu     sync.Mutex
	openaiModelStates = map[string]*openaiModelState{}
)

const (
	openaiInitialBackoff = 30 * time.Second
	openaiMaxBackoff     = 16 * time.Minute
)

func resetOpenAIModelStates() {
	openaiModelMu.Lock()
	defer openaiModelMu.Unlock()
	openaiModelStates = make(map[string]*openaiModelState)
}

func modelStateLocked(name string) *openaiModelState {
	st, ok := openaiModelStates[name]
	if !ok {
		st = &openaiModelState{failureDelay: openaiInitialBackoff}
		openaiModelStates[name] = st
	}
	return st
}

func initAIs() {
	AIs = make([]*AIInstance, 0)
	AIs = append(AIs, &AIInstance{
		Name:   "OpenAI Like",
		Init:   initOpenAI,
		Update: getContentOpenAI,
	})
	for _, ai := range AIs {
		ai.Init(ai)
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				pruneAIHourlyContent(now)
				if aiContentNeedsRefill(now) {
					if err := refreshAIContent(now); err != nil {
						fmt.Printf("AI content refill failed at %s: %v\n", now.Format("15:04:05"), err)
					}
				}
			}
		}
	}(aiContext)
}

func refreshAIContent(t time.Time) error {
	aiGenerationMu.Lock()
	defer aiGenerationMu.Unlock()
	if !aiContentNeedsRefill(t) {
		return nil
	}
	for _, ai := range AIs {
		if ai.Valid {
			if err := ai.Update(ai, t, nil); err == nil {
				fmt.Printf("AI content batch appended at %s\n", t.Format("15:04:05"))
				return nil
			} else {
				return err
			}
		}
	}
	return errors.New("no valid AI provider")
}

func initOpenAI(self *AIInstance) {
	apiKey := GetOpenAIAPIKey()
	if apiKey == "" {
		fmt.Println("OpenAI api key is empty")
		self.Valid = false
		return
	}
	config := openai.DefaultConfig(apiKey)
	if baseURL := GetOpenAIBaseURL(); baseURL != "" {
		config.BaseURL = baseURL
	}
	openaiClient = openai.NewClientWithConfig(config)
	resetOpenAIModelStates()
	if getContentOpenAI(self, time.Now(), nil) == nil {
		self.Valid = true
	} else {
		self.Valid = false
	}
}

// reloadAIConfig 热重载 AI 配置（配置变更后调用）
func reloadAIConfig() {
	fmt.Println("热重载 AI 配置...")
	for _, ai := range AIs {
		if ai.Name == "OpenAI Like" {
			ai.Init(ai)
			if ai.Valid {
				fmt.Println("AI 配置重载成功")
			} else {
				fmt.Println("AI 配置重载失败")
			}
			return
		}
	}
}

func AIContentValid() bool {
	aiContentMu.Lock()
	defer aiContentMu.Unlock()
	return len(aiHourlyContent[hourKey(time.Now())]) > 0
}

func AIContentPop() (result string) {
	aiContentMu.Lock()
	defer aiContentMu.Unlock()
	hour := hourKey(time.Now())
	pool := aiHourlyContent[hour]
	if len(pool) == 0 {
		return ""
	}
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return ""
	}
	resultIdx := int(idx.Int64())
	result = pool[resultIdx]
	aiHourlyContent[hour] = append(pool[:resultIdx], pool[resultIdx+1:]...)
	fmt.Println("AI current-hour pool len:", len(aiHourlyContent[hour]), "result:", result)
	return result
}

func AISampleApped(s string) {
	AISamples = append(AISamples, s)
	for len(AISamples) > 13 {
		AISamples = AISamples[1:]
	}
}

func configuredModels() []string {
	models := GetOpenAIModels()
	if len(models) == 0 {
		return []string{openai.GPT4oMini}
	}
	return models
}

func getContentOpenAI(self *AIInstance, t time.Time, pool *[]string) (err error) {
	models := configuredModels()
	prompt := []openai.ChatCompletionMessage{{
		Role:    openai.ChatMessageRoleUser,
		Content: getPrompt(t),
	}}
	return rotateTryModels(models, func(model string) error {
		entries, err := callOpenAIModel(model, prompt)
		if err != nil {
			return err
		}
		appendAIHourlyContent(entries)
		recordAIResults(entries)
		return nil
	})
}

// rotateTryModels 始终从配置的首个模型开始；仅首选不可用时顺序 fallback。
func rotateTryModels(models []string, call func(string) error) error {
	if len(models) == 0 {
		return errors.New("no models")
	}

	var lastErr error
	tried := 0
	now := time.Now()

	for i := 0; i < len(models); i++ {
		model := models[i]

		openaiModelMu.Lock()
		st := modelStateLocked(model)
		inBackoff := now.Before(st.nextRetry)
		nextAt := st.nextRetry
		openaiModelMu.Unlock()
		if inBackoff {
			fmt.Printf("skip model %s until %s\n", model, nextAt.Format("15:04:05"))
			continue
		}

		tried++
		err := call(model)
		if err == nil {
			openaiModelMu.Lock()
			st = modelStateLocked(model)
			st.failureDelay = openaiInitialBackoff
			st.nextRetry = time.Time{}
			openaiModelMu.Unlock()
			return nil
		}

		lastErr = err
		openaiModelMu.Lock()
		st = modelStateLocked(model)
		st.nextRetry = time.Now().Add(st.failureDelay)
		fmt.Printf("model %s failed, next retry after %s (at %s)\n", model, st.failureDelay, st.nextRetry.Format("15:04:05"))
		st.failureDelay *= 2
		if st.failureDelay > openaiMaxBackoff {
			st.failureDelay = openaiMaxBackoff
		}
		openaiModelMu.Unlock()
	}

	if tried == 0 {
		return errors.New("all models in backoff")
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("all models failed")
}

func callOpenAIModel(model string, prompt []openai.ChatCompletionMessage) ([]AIRecentResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: prompt,
		Stream:   false,
	}

	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	fmt.Printf("即将发送的 JSON Payload:\n%s\n", string(reqJSON))

	resp, err := openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		e := &openai.APIError{}
		if errors.As(err, &e) {
			switch e.HTTPStatusCode {
			case 400:
				fmt.Println("=== 400 Bad Request 详情 ===")
				fmt.Printf("错误代码 (Code): %v\n", e.Code)
				fmt.Printf("错误类型 (Type): %s\n", e.Type)
				fmt.Printf("错误信息 (Message): %s\n", e.Message)
				if e.Param != nil {
					fmt.Printf("相关参数 (Param): %s\n", *e.Param)
				} else {
					fmt.Printf("相关参数 (Param): <null>\n")
				}
			default:
				fmt.Printf("API Error: %v\n", e)
			}
		} else {
			fmt.Printf("Generic Error: %v\n", err)
		}
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("AI returned no choices")
	}
	entries, err := parseAIHourlyResults(resp.Choices[0].Message.Content, time.Now())
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func getPrompt(t time.Time) string {
	start := t.Truncate(time.Hour)
	good := randomAIEntries(aiCuratedEntries("good"), 40)
	for _, sample := range defaultGoodSamples {
		if len(good) >= 40 {
			break
		}
		good = append(good, sample)
	}
	bad := randomAIEntries(aiCuratedEntries("bad"), 10)
	defaultBad := []string{"吃饭", "睡觉", "开心", "努力工作", "一切顺利", "喝水", "刷手机", "出门", "休息", "加油"}
	for _, sample := range defaultBad {
		if len(bad) >= 10 {
			break
		}
		bad = append(bad, sample)
	}

	hours := make([]string, 0, aiBatchHours)
	for i := 0; i < aiBatchHours; i++ {
		hours = append(hours, hourKey(start.Add(time.Duration(i)*time.Hour)))
	}
	return fmt.Sprintf("%s\n当前时间是%s。请为以下每个小时各生成恰好 %d 条词条：%s。\n"+
		"优质风格参考（可学习风格但禁止复用）：%s。\n"+
		"负面示例（禁止模仿其平淡、泛化或无趣的风格）：%s。\n"+
		"只输出 %d 行，严格格式为 [YYYY-MM-DD HH]{词条}；每个所列小时必须有 %d 行。",
		promptDefault, todayChineseDateTime(t), aiEntriesPerHour, strings.Join(hours, "、"),
		wrapAISamples(good), wrapAISamples(bad), aiBatchHours*aiEntriesPerHour, aiEntriesPerHour)
}

func wrapAISamples(samples []string) string {
	out := make([]string, 0, len(samples))
	for _, sample := range samples {
		out = append(out, "{"+sample+"}")
	}
	return strings.Join(out, " ")
}

var aiHourlyResultPattern = regexp.MustCompile(`(?m)^\s*\[(\d{4}-\d{2}-\d{2} \d{2})\]\{([^{}\r\n]+)\}\s*$`)

func parseAIHourlyResults(content string, t time.Time) ([]AIRecentResult, error) {
	expected := make(map[string]bool, aiBatchHours)
	start := t.Truncate(time.Hour)
	for i := 0; i < aiBatchHours; i++ {
		expected[hourKey(start.Add(time.Duration(i)*time.Hour))] = true
	}
	perHour := make(map[string][]AIRecentResult, aiBatchHours)
	seen := make(map[string]bool)
	for _, match := range aiHourlyResultPattern.FindAllStringSubmatch(content, -1) {
		hour, text := match[1], strings.TrimSpace(match[2])
		if !expected[hour] || text == "" || len([]rune(text)) > 64 || seen[hour+"\x00"+text] {
			continue
		}
		seen[hour+"\x00"+text] = true
		if len(perHour[hour]) < aiEntriesPerHour {
			perHour[hour] = append(perHour[hour], AIRecentResult{
				Text: text, Hour: hour, GeneratedAt: time.Now().Format(gTimeFormat),
			})
		}
	}
	entries := make([]AIRecentResult, 0, aiBatchHours*aiEntriesPerHour)
	for hour := range expected {
		if len(perHour[hour]) < aiEntriesPerHour {
			return nil, fmt.Errorf("AI returned only %d valid entries for %s", len(perHour[hour]), hour)
		}
		entries = append(entries, perHour[hour]...)
	}
	return entries, nil
}
