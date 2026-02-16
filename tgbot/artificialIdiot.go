package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"math/rand/v2"

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

var promptDefault = "你是一个算命机器人，会随机给出一些老黄历词条，类似：[宜xxxxxxx][忌xxxxxxx]，词条范围包含但不限于与[今天的日期]、[现在的时间]相关的活动、[Ingress]游戏中的行为、衣服穿搭、发型发色、交通工具、饮食搭配、经典网络迷因、流行搞笑梗等等各种有趣的东西；其中，Ingress中的[名词]均使用英文。词条必须简短不含逗号，但是需要搞笑有趣、幽默讽刺。当今天是节日时，生成词条尽量与节日相关，生成词条不含“宜”或“忌”的前缀，但是需要保证词条加上“宜”或“忌”的前缀时都意思通顺，生成的词条均以大括号{}括住，请参考后面的生成词条示例，再生成13条词条。\n"
var promptEnd = "请仅回答生成的词条，每个词条一行。"

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

var AIContentPool []string
var AINextHourPool []string
var AIs []*AIInstance

func AIContentPush(pool *[]string, s string) {
	*pool = append(*pool, s)
	for len(*pool) > 20 {
		*pool = (*pool)[1:]
	}
}

var aiContext, cancelAI = context.WithCancel(context.Background())

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

		lastHour := time.Now().Hour()
		failureDelay := 30 * time.Second
		var nextRegularUpdate time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				currentHour := now.Hour()
				currentMinute := now.Minute()

				// 1. 整点切换逻辑
				if currentHour != lastHour {
					fmt.Printf("New hour detected: %d (was %d). Switching AIContentPool.\n", currentHour, lastHour)
					if len(AINextHourPool) > 0 {
						AIContentPool = AINextHourPool
						AINextHourPool = make([]string, 0)
					} else {
						// 如果预取失败了，整点至少清空旧的（过期的）
						AIContentPool = make([]string, 0)
					}
					lastHour = currentHour
				}

				// 2. 触发决策逻辑
				var targetPool *[]string
				var targetTime time.Time
				isPrefetch := false

				if currentMinute == 59 {
					// 59分进入预取模式
					if len(AINextHourPool) < 5 {
						targetPool = &AINextHourPool
						targetTime = now.Add(time.Hour)
						isPrefetch = true
					}
				} else {
					// 常规模式：池不满 且 预取池为空 且 过了延迟时间
					if len(AIContentPool) < 5 && len(AINextHourPool) == 0 && now.After(nextRegularUpdate) {
						targetPool = &AIContentPool
						targetTime = now
					}
				}

				// 3. 执行更新
				if targetPool != nil {
					var success bool
					for _, ai := range shuffle(AIs) {
						if ai.Valid {
							if err := ai.Update(ai, targetTime, targetPool); err == nil {
								success = true
								break
							}
						}
					}

					if success {
						fmt.Printf("AI content updated successfully (Prefetch: %v) at %s\n", isPrefetch, now.Format("15:04:05"))
						failureDelay = 30 * time.Second
						nextRegularUpdate = time.Time{} // 重置延迟
					} else {
						if !isPrefetch {
							// 只有常规模式失败才增加延迟
							fmt.Printf("Regular AI update failed at %s, retrying after %s\n", now.Format("15:04:05"), failureDelay)
							nextRegularUpdate = now.Add(failureDelay)
							failureDelay *= 2
							if failureDelay > 16*time.Minute {
								failureDelay = 16 * time.Minute
							}
						} else {
							fmt.Printf("Prefetch AI update failed at %s, will retry soon\n", now.Format("15:04:05"))
						}
					}
				}
			}
		}
	}(aiContext)
}

func shuffle(arr []*AIInstance) []*AIInstance {
	rand.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
	return arr
}

func initOpenAI(self *AIInstance) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OpenAI api key is empty")
		self.Valid = false
		return
	}
	config := openai.DefaultConfig(apiKey)
	if os.Getenv("OPENAI_BASE_URL") != "" {
		config.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}
	openaiClient = openai.NewClientWithConfig(config)
	if getContentOpenAI(self, time.Now(), &AIContentPool) == nil {
		self.Valid = true
	} else {
		self.Valid = false
	}
}

func AIContentValid() bool {
	return len(AIContentPool) > 0
}

func AIContentPop() (result string) {
	resultIdx := rand.IntN(len(AIContentPool))
	result = AIContentPool[resultIdx]
	AIContentPool = append(AIContentPool[:resultIdx], AIContentPool[resultIdx+1:]...)
	fmt.Println("len:", len(AIContentPool), "result:", result)
	return result
}

func AISampleApped(s string) {
	AISamples = append(AISamples, s)
	for len(AISamples) > 13 {
		AISamples = AISamples[1:]
	}
}

func getContentOpenAI(self *AIInstance, t time.Time, pool *[]string) (err error) {
	prompt := make([]openai.ChatCompletionMessage, 0)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: getPrompt(t),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4oMini,
		Messages: prompt,
		Stream:   false,
	}
	if os.Getenv("OPENAI_MODEL") != "" {
		req.Model = os.Getenv("OPENAI_MODEL")
	}

	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	fmt.Printf("即将发送的 JSON Payload:\n%s\n", string(reqJSON))

	resp, err := openaiClient.CreateChatCompletion(
		ctx,
		req,
	)
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
					// 如果指针不为空，则解引用打印具体值
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
		return
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	re := regexp.MustCompile(`\{(.+?)\}`)
	for _, v := range lines {
		match := re.FindStringSubmatch(v)
		if len(match) > 0 {
			AIContentPush(pool, match[1])
			fmt.Println(self.Name, "AI result add:", match[1], "to pool size:", len(*pool))
		}
	}
	return err
}

func getPrompt(t time.Time) string {
	sample := ""
	for _, v := range AISamples {
		sample += "{" + v + "} "
	}
	fmt.Println("generate AI result with:\n", sample)

	p := promptDefault + sample + "\n现在是" + todayChineseDateTime(t) + "，" + promptEnd
	return p
}
