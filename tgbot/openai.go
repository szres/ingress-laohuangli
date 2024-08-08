package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/exp/rand"
)

var gptClient *openai.Client
var gptLaohuangli []string
var gptSample []string

func initOpenAI() {
	if gAPIKey == "" {
		fmt.Println("gAPIKey is empty")
		return
	}
	gptConfig := openai.DefaultConfig(gAPIKey)
	gptClient = openai.NewClientWithConfig(gptConfig)
	gptLaohuangli = make([]string, 0)
	gptSample = make([]string, 0)

	moreLaohuangliFromAI()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			if len(gptLaohuangli) < 12 {
				moreLaohuangliFromAI()
			}
		}
	}()
}

func gptLaohuangliValid() bool {
	return len(gptLaohuangli) > 0
}

func gptLaohuangliPop() string {
	resultIdx := rand.Intn(len(gptLaohuangli))
	result := gptLaohuangli[resultIdx]
	gptLaohuangli = append(gptLaohuangli[:resultIdx], gptLaohuangli[resultIdx+1:]...)

	randomIdx := rand.Intn(len(gptLaohuangli))
	gptLaohuangli = append(gptLaohuangli[:randomIdx], gptLaohuangli[randomIdx+1:]...)
	return result
}

func gptSampleApped(s string) {
	gptSample = append(gptSample, s)
	for len(gptSample) > 5 {
		gptSample = gptSample[1:]
	}
}

func moreLaohuangliFromAI() {
	prompt := make([]openai.ChatCompletionMessage, 0)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你是一个游戏Ingress群组中的随机算命机器人，随机给出形如\"宜xxx，忌xxx。\"的结果。结果词条可能包含衣服穿搭、发型发色、交通工具、饮食搭配、经典网络迷因和Ingress游戏中的行为等；其中，Ingress中的名词均使用英文。结果词条尽量搞笑有趣或是幽默讽刺，宜和忌的词条尽量无关。生成文字中仅包含结果。如果理解请生成5个示例词条。在示例词条之后，我说一个数字，你就再生成对应条数的结果。",
	})

	sample := ""
	if len(gptSample) != 5 {
		sample = "1. 宜拒接领导电话，忌翘班去钓鱼。\n2. 宜投食减肥者，忌变得不幸。\n3. 宜刷AP，忌喝胡萝卜玉米猪骨汤。\n4. 宜橙色针织裙，忌搭星舰去上班。\n5. 宜痛击队友，忌把小鹿撞晕。"
	} else {
		for i, v := range gptSample {
			sample += strconv.Itoa(i+1) + ". " + v + "\n"
		}
	}
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: sample,
	})
	fmt.Println("generate AI result with:", sample)

	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "10",
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
				gptLaohuangli = append(gptLaohuangli, match[0])
				fmt.Println("AI result add:", match[0])
			}
		}
	}
}
