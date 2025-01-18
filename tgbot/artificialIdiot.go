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

var promptDefault = "你是一个算命机器人，会随机给出一个今天的幸运物或者行为词条，词条范围包含但不限于与今天日期相关的节日活动、Ingress游戏中的行为、衣服穿搭、发型发色、交通工具、饮食搭配、经典网络迷因等等各种有趣的东西；其中，Ingress中的名词均使用英文。词条必须简短不含逗号，但是需要搞笑有趣、幽默讽刺。当今天是节日时，生成词条尽量与节日相关，生成词条均以大括号{}括住，请参考后面的生成词条示例，再生成10条词条。\n"
var promptEnd = "请仅回答生成的词条，每个词条一行。"

func todayChineseDate() string {
	t := time.Now()
	return fmt.Sprintf("%d年%d月%d日", t.Year(), t.Month(), t.Day())
}

type AIInstance struct {
	Name   string
	Init   func(*AIInstance)
	Update func(*AIInstance)
	Pool   []string
}

var AIs []*AIInstance

func initAIs() {
	AIs = make([]*AIInstance, 0)
	AIs = append(AIs, &AIInstance{
		Name:   "GPT4oMini",
		Init:   initOpenAI,
		Update: getContentOpenAI,
	})
	AIs = append(AIs, &AIInstance{
		Name:   "Gemini2.0flash",
		Init:   initGemini,
		Update: getContentGemini,
	})
	for _, ai := range AIs {
		ai.Init(ai)
	}
}

func initGemini(self *AIInstance) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Gemini api key is empty")
		return
	}
	self.Pool = make([]string, 0)

	getContentGemini(self)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			if len(self.Pool) < 10 {
				getContentGemini(self)
			}
		}
	}()
}

func getContentGemini(self *AIInstance) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := gemini.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash-exp")
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
						self.Pool = append(self.Pool, match[1])
						fmt.Println(self.Name, "AI result add:", match[1])
					}
					for len(self.Pool) >= 15 {
						self.Pool = self.Pool[1:]
					}
				}
			}
		}
	}

}

func initOpenAI(self *AIInstance) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OpenAI api key is empty")
		return
	}
	self.Pool = make([]string, 0)

	gptClient = openai.NewClientWithConfig(openai.DefaultConfig(apiKey))

	getContentOpenAI(self)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			if len(self.Pool) < 10 {
				getContentOpenAI(self)
			}
		}
	}()
}

func AIContentValid() bool {
	sum := 0
	for _, v := range AIs {
		sum += len(v.Pool)
	}
	return sum > 0
}

func AIContentPop() (result string, name string) {
	aiIdx := rand.Intn(len(AIs))
	if len(AIs[aiIdx].Pool) == 0 {
		return "", ""
	}
	resultIdx := rand.Intn(len(AIs[aiIdx].Pool))
	result = AIs[aiIdx].Pool[resultIdx]
	AIs[aiIdx].Pool = append(AIs[aiIdx].Pool[:resultIdx], AIs[aiIdx].Pool[resultIdx+1:]...)
	fmt.Println(AIs[aiIdx].Name, "len:", len(AIs[aiIdx].Pool), "result:", result)
	return result, AIs[aiIdx].Name
}

func AISampleApped(s string) {
	AISamples = append(AISamples, s)
	for len(AISamples) > 13 {
		AISamples = AISamples[1:]
	}
}

func getContentOpenAI(self *AIInstance) {
	prompt := make([]openai.ChatCompletionMessage, 0)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: getPrompt(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := gptClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			MaxTokens: 1024,
			Model:     openai.GPT4oMini,
			Messages:  prompt,
		},
	)
	if err != nil {
		fmt.Println("Error", err)
	} else {
		lines := strings.Split(resp.Choices[0].Message.Content, "\n")
		re := regexp.MustCompile(`\{宜(.+?)\}`)
		for _, v := range lines {
			match := re.FindStringSubmatch(v)
			if len(match) > 0 {
				self.Pool = append(self.Pool, match[1])
				fmt.Println(self.Name, "AI result add:", match[1])
			}
			for len(self.Pool) >= 15 {
				self.Pool = self.Pool[1:]
			}
		}
	}
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
