package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	gemini "github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/exp/rand"
	"google.golang.org/api/option"
)

var gptClient *openai.Client
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

var promptDefault = "你是一个算命机器人，会随机给出一个今天的幸运物或者行为词条，词条范围包含但不限于与今天日期相关的节日活动、Ingress游戏中的行为、衣服穿搭、发型发色、交通工具、饮食搭配、经典网络迷因等等各种有趣的东西；其中，Ingress中的名词均使用英文。词条必须简短不含逗号，但是需要搞笑有趣、幽默讽刺。当今天是节日时，生成词条尽量与节日相关，生成词条均以大括号{}括住，请参考后面的生成词条示例，再生成13条词条。\n"
var promptEnd = "请仅回答生成的词条，每个词条一行。"

func todayChineseDate() string {
	t := time.Now()
	return fmt.Sprintf("%d年%d月%d日", t.Year(), t.Month(), t.Day())
}

type AIInstance struct {
	Name   string
	Init   func(*AIInstance)
	Update func(*AIInstance) error
	Valid  bool
}

var AIContentPool []string
var AIs []*AIInstance

func AIContentPush(s string) {
	AIContentPool = append(AIContentPool, s)
	for len(AIContentPool) > 15 {
		AIContentPool = AIContentPool[1:]
	}
}

func initAIs() {
	AIs = make([]*AIInstance, 0)
	AIs = append(AIs, &AIInstance{
		Name:   "GPT4oMini",
		Init:   initOpenAI,
		Update: getContentOpenAI,
	})
	AIs = append(AIs, &AIInstance{
		Name:   "Gemini2.5flash",
		Init:   initGemini,
		Update: getContentGemini,
	})
	for _, ai := range AIs {
		ai.Init(ai)
	}

	go func() {
		var delay time.Duration = 30 * time.Second
		for {
			var updated bool
			if len(AIContentPool) < 5 {
				for _, ai := range shuffle(AIs) {
					if ai.Valid {
						if err := ai.Update(ai); err == nil {
							updated = true
							break
						}
					}
				}
				if !updated {
					// HH:MM:SS logging
					fmt.Printf("All AIs content failed to update at %s, retrying in %s...\n", time.Now().Format("15:04:05"), delay)
					delay *= 2 // Increase delay by 2 times if all AIs fail
					if delay > 16*time.Minute {
						delay = 16 * time.Minute // Cap the delay at 16 minutes
					}
				} else {
					fmt.Println("AI content updated successfully at", time.Now().Format("15:04:05"))
					delay = 30 * time.Second // Reset delay to 30 seconds after a successful update
				}
			}
			<-time.After(delay)
		}
	}()
}

func shuffle(arr []*AIInstance) []*AIInstance {
	rand.Seed(uint64(time.Now().UnixNano()))
	rand.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
	return arr
}

func initGemini(self *AIInstance) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Gemini api key is empty")
		self.Valid = false
		return
	}

	if getContentGemini(self) == nil {
		self.Valid = true
	} else {
		self.Valid = false
	}
}

func getContentGemini(self *AIInstance) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var client *gemini.Client
	if os.Getenv("GEMINI_BASE_URL") != "" {
		client, err = gemini.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")), option.WithEndpoint(os.Getenv("GEMINI_BASE_URL")))
	} else {
		client, err = gemini.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	}
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	defer client.Close()
	var model *gemini.GenerativeModel
	if os.Getenv("GEMINI_MODEL") != "" {
		model = client.GenerativeModel(os.Getenv("GEMINI_MODEL"))
	} else {
		model = client.GenerativeModel("gemini-2.5-flash-preview-05-20")
	}
	resp, err := model.GenerateContent(ctx, gemini.Text(getPrompt()))
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				lines := strings.Split(fmt.Sprint(part), "\n")
				re := regexp.MustCompile(`\{宜(.+?)\}`)
				for _, v := range lines {
					match := re.FindStringSubmatch(v)
					if len(match) > 0 {
						AIContentPush(match[1])
						fmt.Println(self.Name, "AI result add:", match[1])
					}
				}
			}
		}
	}
	return err
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
	gptClient = openai.NewClientWithConfig(config)
	if getContentOpenAI(self) == nil {
		self.Valid = true
	} else {
		self.Valid = false
	}
}

func AIContentValid() bool {
	return len(AIContentPool) > 0
}

func AIContentPop() (result string) {
	resultIdx := rand.Intn(len(AIContentPool))
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

func getContentOpenAI(self *AIInstance) (err error) {
	prompt := make([]openai.ChatCompletionMessage, 0)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: getPrompt(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req := openai.ChatCompletionRequest{
		MaxTokens: 1024,
		Model:     openai.GPT4oMini,
		Messages:  prompt,
	}
	if os.Getenv("OPENAI_MODEL") != "" {
		req.Model = os.Getenv("OPENAI_MODEL")
	}
	resp, err := gptClient.CreateChatCompletion(
		ctx,
		req,
	)
	if err != nil {
		fmt.Println("Error", err)
		return
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	re := regexp.MustCompile(`\{宜(.+?)\}`)
	for _, v := range lines {
		match := re.FindStringSubmatch(v)
		if len(match) > 0 {
			AIContentPush(match[1])
			fmt.Println(self.Name, "AI result add:", match[1])
		}
	}
	return err
}

func getPrompt() string {
	sample := ""
	for _, v := range AISamples {
		sample += "{宜" + v + "} "
	}
	fmt.Println("generate AI result with:\n", sample)

	p := promptDefault + sample + "\n今天是" + todayChineseDate() + "，" + promptEnd
	return p
}
