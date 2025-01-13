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
var AISamples []string

var sampleDefault = "宜拒接领导电话，忌翘班去钓鱼。\n宜投食减肥者，忌变得不幸。\n宜刷AP，忌喝胡萝卜玉米猪骨汤。\n宜橙色针织裙，忌搭星舰去上班。\n宜痛击队友，忌把小鹿撞晕。"
var promptDefault = "你是一个Ingress游戏群组中的随机算命机器人，随机给出形如\"宜xxx，忌xxx。\"的结果。结果词条可以包含与当前日期相关的节日活动、衣服穿搭、发型发色、交通工具、饮食搭配、经典网络迷因和Ingress游戏中的行为等；其中，Ingress中的名词均使用英文，包含ingress行为的词条不要超过一半。结果尽量搞笑有趣、幽默讽刺。以下是5个参考示例词条。请再生成10条结果每个结果单独一行，生成文字中仅包含结果。\n"

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
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			if len(self.Pool) < 5 {
				self.Pool = make([]string, 0)
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
				re := regexp.MustCompile(`宜.+?，忌.+?。`)
				for _, v := range lines {
					match := re.FindStringSubmatch(v)
					if len(match) > 0 {
						self.Pool = append(self.Pool, match[0])
						fmt.Println(self.Name, "AI result add:", match[0])
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
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			if len(self.Pool) < 5 {
				self.Pool = make([]string, 0)
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
	resultIdx := rand.Intn(len(AIs))
	if len(AIs[resultIdx].Pool) == 0 {
		return "", ""
	}
	result = AIs[resultIdx].Pool[rand.Intn(len(AIs[resultIdx].Pool))]
	AIs[resultIdx].Pool = append(AIs[resultIdx].Pool[:resultIdx], AIs[resultIdx].Pool[resultIdx+1:]...)
	return result, AIs[resultIdx].Name
}

func AISampleApped(s string) {
	AISamples = append(AISamples, s)
	for len(AISamples) > 5 {
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
		re := regexp.MustCompile(`宜.+?，忌.+?。`)
		for _, v := range lines {
			match := re.FindStringSubmatch(v)
			if len(match) > 0 {
				self.Pool = append(self.Pool, match[0])
				fmt.Println(self.Name, "AI result add:", match[0])
			}
		}
	}
}

func getPrompt() string {
	sample := ""
	if len(AISamples) < 5 {
		sample = sampleDefault
	} else {
		for _, v := range AISamples {
			sample += v + "\n"
		}
	}
	fmt.Println("generate AI result with:\n", sample)

	p := "今天是" + todayChineseDate() + "，" + promptDefault + sample
	return p
}
