package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/adrg/strutil/metrics"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	scribble "github.com/nanobox-io/golang-scribble"
	tele "gopkg.in/telebot.v3"
)

//go:embed dist/*
var staticFS embed.FS

type testenv struct {
	Token     string `json:"token"`
	AdminID   string `json:"adminid"`
	KumaURL   string `json:"kumaurl"`
	OpenAiKey string `json:"openaikey"`
}

var testEnv testenv

type entry struct {
	UUID      string `json:"uuid"`
	Content   string `json:"content"`
	Nominator string `json:"nominator"`
}
type laohuangliTemplate struct {
	Desc   string   `json:"desc"`
	Values []string `json:"values"`
}
type laohuangliResult struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

var (
	gTimeFormat  string = "2006-01-02 15:04"
	gAdminID     int64
	gKumaPushURL string
	gToken       string

	gStrCompareAlgo *metrics.Jaro
)

var db *scribble.Driver

func SetupApp() {
	db, _ = scribble.New("../db", nil)

	// 初始化日志缓冲
	InitLogBuffer(1000)

	// 加载配置（优先 DB，否则从环境变量初始化）
	loadConfig()

	// 加载 API Token 和用户统计缓存
	loadAPITokens()
	loadUserStats()

	// 从配置中读取运行时变量
	gToken = GetBotToken()
	gAdminID = GetAdminID()
	gKumaPushURL = GetKumaPushURL()

	laoHL.init(db)
	laoHL.start()

	gStrCompareAlgo = metrics.NewJaro()
	gStrCompareAlgo.CaseSensitive = false

	// 初始化 AI
	initAIs()
}

var b *tele.Bot

func fullName(u *tele.User) string {
	if u.FirstName == "" || u.LastName == "" {
		return u.FirstName + u.LastName
	}
	return u.FirstName + " " + u.LastName
}

// restartBot 停止旧 bot 并重新启动（用于 token/管理员变更后热重载）
func restartBot() bool {
	if b != nil {
		fmt.Println("正在停止旧 Telegram Bot...")
		b.RemoveWebhook(true)
		b = nil
	}
	return startBot()
}

// handleWebhook 处理 Telegram Webhook 推送
func handleWebhook(c fiber.Ctx) error {
	// 验证 URL 中的 token 与当前 bot token 匹配
	urlToken := c.Params("token")
	if urlToken != gToken {
		return c.SendStatus(fiber.StatusNotFound)
	}

	var update tele.Update
	body := c.Body()
	if err := json.Unmarshal(body, &update); err != nil {
		fmt.Println("Webhook 解析失败:", err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	b.ProcessUpdate(update)
	return c.SendStatus(fiber.StatusOK)
}

func startBot() bool {
	gToken = GetBotToken()
	gAdminID = GetAdminID()
	if gToken == "" {
		fmt.Println("⚠️ BOT_TOKEN 未设置，请通过 Web 管理面板配置后重启服务")
		return false
	}

	pref := tele.Settings{
		Token:  gToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	var err error
	b, err = tele.NewBot(pref)
	if err != nil {
		fmt.Println("⚠️ Telegram Bot 启动失败:", err)
		return false
	}

	for _, s := range chatCMD {
		b.Handle(s, func(c tele.Context) error {
			return cmdInChatHandler(c)
		})
	}

	b.Handle(tele.OnText, func(c tele.Context) error {
		return msgInChatHandler(c)
	})

	mk := &tele.ReplyMarkup{ResizeKeyboard: true}
	voteApproveBtn := mk.Data("赞成", "voteApproveBtn")
	voteRefuseBtn := mk.Data("反对", "voteRefuseBtn")
	deleteBtn := mk.Data("删除", "deleteBtn")
	b.Handle(&voteApproveBtn, voteApprove())
	b.Handle(&voteRefuseBtn, voteRefuse())
	b.Handle(&deleteBtn, msgDelete())

	b.Handle(tele.OnQuery, func(c tele.Context) error {
		results := make(tele.Results, 0)
		for _, v := range nominations {
			if v.NominatorID == c.Sender().ID {
				results = append(results, buildVotes(v))
			}
		}
		annualYear := getAnnualYear()
		if annualYear > 0 && len(laoHL.annualSummary(c.Sender().ID)) > 0 {
			annualTitle := fmt.Sprintf("%d年度总结", annualYear)
			results = append(results, &tele.ArticleResult{
				Title: annualTitle,
				Text:  fullName(c.Sender()) + " " + annualTitle + ":\n" + laoHL.annualSummary(c.Sender().ID),
			})
		}
		results = append(results, &tele.ArticleResult{
			Title: "今日我的老黄历",
			Text:  fullName(c.Sender()) + " " + laoHL.randomToday(c.Sender().ID, fullName(c.Sender())),
		})
		return c.Answer(&tele.QueryResponse{
			Results:           results,
			CacheTime:         3,
			IsPersonal:        true,
			SwitchPMText:      "提名新词条",
			SwitchPMParameter: "nominate",
		})
	})

	// 注册 Webhook（如果有域名配置）
	webDomain := GetWebDomain()
	if webDomain != "" {
		webhookURL := "https://" + webDomain + "/webhook/" + gToken
		err = b.SetWebhook(&tele.Webhook{
			Endpoint: &tele.WebhookEndpoint{PublicURL: webhookURL},
			Listen:   "", // 不自己启动 HTTP server
		})
		if err != nil {
			fmt.Println("⚠️ Webhook 注册失败，降级为 Long Polling:", err)
		} else {
			fmt.Println("Telegram Webhook 已注册:", webhookURL)
		}
	} else {
		fmt.Println("WEB_DOMAIN 未配置，使用 Long Polling 模式")
		go b.Start()
	}

	fmt.Println("Telegram Bot 已启动")
	return true
}

func main() {
	SetupApp()
	NominationInit()
	fmt.Println("老黄历启动！")

	// 创建 Fiber app
	app := fiber.New()

	// 注册 API + Webhook 路由（更具体的路由优先匹配）
	SetupRoutes(app)

	// 提取嵌入的 dist 子目录
	distFS, err := fs.Sub(staticFS, "dist")
	if err != nil {
		fmt.Println("⚠️ 无法加载嵌入的前端文件:", err)
	} else {
		// 静态文件服务 + SPA fallback
		// static.New 从 FS 根路径 "/" 查找文件，NotFoundHandler 处理 SPA 路由
		app.Get("/*", static.New("", static.Config{
			FS: distFS,
			NotFoundHandler: func(c fiber.Ctx) error {
				c.Set("Content-Type", "text/html; charset=utf-8")
				f, err := distFS.Open("index.html")
				if err != nil {
					return c.SendStatus(fiber.StatusNotFound)
				}
				defer f.Close()
				data, err := io.ReadAll(f)
				if err != nil {
					return c.SendStatus(fiber.StatusInternalServerError)
				}
				return c.Send(data)
			},
		}))
	}

	// 尝试启动 Telegram Bot
	botStarted := startBot()
	if !botStarted {
		fmt.Println("服务已启动（仅 Web 管理模式），请通过管理面板配置 BOT_TOKEN 后重启服务")
	}

	// 启动 Fiber HTTP 服务
	go func() {
		fmt.Println("Fiber HTTP 服务启动，监听 :80")
		if err := app.Listen(":80", fiber.ListenConfig{
			DisableStartupMessage: true,
		}); err != nil {
			panic(err)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	<-sc
	if b != nil {
		b.RemoveWebhook(true)
	}
	fmt.Println("由于即将关闭，进行数据备份")
	laoHL.save()
	<-time.After(time.Second * 1)
	fmt.Println("下线！")
}
