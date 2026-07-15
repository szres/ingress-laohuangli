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
	// Ingress 向（术语用法必须真实可信）
	"边砍圣诞树边刷AP", "带猫猫参加IFS", "给Scanner贴钢化膜",
	"把XMP当烟花", "用ADA处理前任", "为一条跨城Link绕路三公里", "给Scanner充电到忘记睡觉",
	"骑共享单车追稀有Portal", "绕远路收集XM", "把通勤路线画成Field", "和敌对阵营拼桌",
	"在雨里更新Scanner", "把钥匙串当护身符", "对着P8许愿",
	"把午休献给Portal", "在地铁里规划大三角", "为一根Link熬夜", "把IFS当相亲局",
	"用Jarvis解决选择困难", "给外卖备注阵营色", "在奶茶里加抵抗",
	"把加班解释为刷AP", "冒雨出门补Resonator", "给闹钟设置成Scanner提示音",
	// 非 Ingress 向（日常恶搞与流行梗）
	"拒接领导电话", "翘班去钓鱼", "投食减肥者", "变得不幸",
	"橙色针织裙", "搭星舰去上班", "胡萝卜玉米猪骨汤",
	"把低电量当人生哲学", "穿拖鞋参加战术会议", "假装看不见群消息",
	"被风吹乱战术发型", "对着地图假装很忙", "给鞋带打战术结",
	"把迷路称为实地勘测", "在凌晨维护社交能量", "在工位假装听懂了",
	"给周一发好人卡", "对镜子练习已读不回", "把体检报告当盲盒拆",
	"给猫排队道歉", "在地铁抢到座位却坐过站", "用外卖红包决定晚餐",
}

var defaultBadSamples = []string{
	// 平淡泛化
	"吃饭", "睡觉", "开心", "努力工作", "一切顺利", "喝水", "刷手机", "出门", "休息", "加油",
	// 术语使用错误（Portal 不可移动、Link 是虚拟的、XMP 是一次性武器）
	"捡起一个Portal带回家", "在Link上晾衣服", "给XMP充电",
}

var promptDefault = `你是「Ingress老黄历」的词条生成器，为 Ingress 玩家生成每日算命用的黄历词条。

【输出格式】
- 每条词条是一个可以直接接在“宜”或“忌”之后的短语，两种接法都必须自然通顺。
- 不含“宜”“忌”前缀，不含逗号、大括号、引号和句号，不超过 64 个 Unicode 字符。
- 不要复述示例，同一批内不得重复或高度相似。

【题材配比（硬性要求）】
- 每个小时的词条中，至少 1/3 必须与 Ingress 完全无关，一个 Ingress 术语都不能出现。
- 无关条目写日常生活恶搞、职场社畜梗、网络流行梗、食物、穿搭、天气、交通等。

【时间贴合（硬性要求）】
- 每个小时的词条要贴合该小时的典型生活场景：清晨写早餐/通勤/起床气，中午写午饭/午休，下午写摸鱼/下午茶/犯困，傍晚写下班/晚饭，深夜写熬夜/夜宵/失眠，凌晨写梦游/赶末班车后的绝望等。
- 严禁时间错位：下午的词条不能出现早餐，早晨的词条不能出现夜宵，工作日白天才有摸鱼梗。
- 注意所列小时的日期是工作日还是周末，周末不写通勤打卡类内容。
- 如果所列日期恰逢中国法定节假日、二十四节气、传统节日或知名网络节日（如程序员节、双十一），当天应有部分词条与之相关。

【Ingress 术语表】涉及 Ingress 的词条必须符合以下真实语义，可以夸张搞笑但不能张冠李戴；专有名词一律用英文：
- Scanner：游戏 App 本体，运行时费电费流量。
- Portal：绑定现实地标（雕塑、涂鸦、建筑等）的虚拟据点，位置固定，玩家要走到附近才能操作。
- hack：对 Portal 取物资的动作，有冷却时间；Glyph Hack 是画符号小游戏，画得好物资更多。
- Resonator：部署在 Portal 上的共振器，共 8 个槽位；全部被打掉后 Portal 变中立可被占领。
- Recharge：消耗 XM 给 Resonator 补能量，持有对应 Portal Key 时可远程充电。
- XM：散布在地图上的能量物质，路过自动收集，Scanner 的一切操作都消耗 XM。
- XMP Burster：一次性攻击武器，用来炸敌方 Resonator；Ultra Strike 是小范围精准版。
- Portal Key：Portal 的钥匙，是连 Link 和远程充电的前提。
- Link：用 Key 在两个己方 Portal 之间连出的能量线，不同 Link 不能交叉；它是虚拟的线，人不能碰到。
- Field：三条 Link 围成的三角形控制场，按覆盖人口获得 MU；超大 Field 俗称 BAF。
- AP：经验值；等级上限 L16，满级后可 Recursion（转生）。
- 阵营：蓝色 Resistance（抵抗军）与绿色 Enlightened（启蒙军）。
- ADA Refactor：把 Portal 翻转成蓝色；JARVIS Virus：把 Portal 翻转成绿色。
- Drone：无人机，可远程逐格移动并 hack。
- Mission：任务，完成后获得拼图奖章；Sojourner：连续每日 hack 的奖章。
- IFS（First Saturday）：每月第一个周六的官方线下聚会；Anomaly：官方大型线下对抗活动。

【风格】
- 简短、荒诞、有讽刺感；幽默来自生活观察和错位联想，不靠堆砌术语。
- 好词条画面感强且具体（如“把通勤路线画成Field”），坏词条平淡空泛（如“努力工作”）。
`

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

// 注入上限：good 大而全（正面示例越多越好），bad 刻意压小——
// 反例数量大了会污染上下文（模型可能学走错误搭配），术语正确性靠术语表兜底。
const (
	aiGoodSampleCap = 60
	aiBadSampleCap  = 12
)

func getPrompt(t time.Time) string {
	start := t.Truncate(time.Hour)
	// 默认样本始终全量保留（分布是人工配平的），标注池随机抽样到上限。
	good := uniqueAITexts(append(randomAIEntries(aiCuratedEntries("good"), aiGoodSampleCap), defaultGoodSamples...))
	bad := uniqueAITexts(append(randomAIEntries(aiCuratedEntries("bad"), aiBadSampleCap), defaultBadSamples...))
	recent := recentAIResults()
	noRepeat := make([]string, 0, len(recent))
	for _, r := range recent {
		noRepeat = append(noRepeat, r.Text)
	}

	hours := make([]string, 0, aiBatchHours)
	for i := 0; i < aiBatchHours; i++ {
		hours = append(hours, hourKey(start.Add(time.Duration(i)*time.Hour)))
	}
	nonIngressMin := (aiEntriesPerHour + 2) / 3
	return fmt.Sprintf("%s\n当前时间是%s。请为以下每个小时各生成恰好 %d 条词条：%s。\n"+
		"优质示例（学习其风格和幽默方式，禁止复用原句）：%s。\n"+
		"劣质示例（包含风格平淡泛化的和 Ingress 术语使用错误的，均禁止模仿）：%s。\n"+
		"最近已生成过的词条（禁止重复或高度相似）：%s。\n"+
		"只输出 %d 行，严格格式为 [YYYY-MM-DD HH]{词条}；每个所列小时必须有 %d 行，其中至少 %d 行完全不含 Ingress 元素。",
		promptDefault, todayChineseDateTime(t), aiEntriesPerHour, strings.Join(hours, "、"),
		wrapAISamples(good), wrapAISamples(bad), wrapAISamples(noRepeat),
		aiBatchHours*aiEntriesPerHour, aiEntriesPerHour, nonIngressMin)
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
