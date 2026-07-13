# AGENTS.md — Ingress老黄历 项目指南

## 项目 DNA

**Ingress老黄历** 是一个 Ingress 游戏主题的"每日算命" Telegram Bot + 配套网站，由老黄历Go分叉而来。

### 架构概览

```
├── tgbot/          # Go 后端 — Telegram Bot + API 服务 + 静态前端托管 (package main)
├── website/        # SvelteKit 前端 — SPA 模式，构建后嵌入 Go 二进制
├── db/             # scribble JSON 文件数据库 (运行时生成)
│   ├── datas/      # 词条库、模板、提名、缓存、streak
│   └── history/    # 每日历史快照
├── Dockerfile      # 多阶段构建：前端 + Go 后端
├── docker-compose.yml  # 单容器编排
└── Makefile        # docker compose 快捷命令
```

### 核心技术栈

| 层 | 技术 |
|---|---|
| **Bot 后端** | **Go 1.25** · **Fiber v3** (HTTP + 静态文件) · `telebot.v3` (Telegram Webhook) · `golang-scribble` (flat-file JSON DB) · `fasttemplate` (`{{var}}` 模板引擎) · `go-openai` · `crypto/rand` |
| **前端** | **SvelteKit 2** · Svelte 4 · **TailwindCSS + DaisyUI** · `@sveltejs/adapter-static` (SPA 模式) |
| **部署** | **单容器 Docker** · `.env` 配置 · Makefile 管理生命周期 · Caddy 反代 (HTTPS) |

### 关键模块

- **`laohuangli.go`** — 核心引擎：词条库加载、加权均衡随机抽取、模板渲染、每日缓存、今日指引生成
- **`artificialIdiot.go`** — AI 内容生成：OpenAI API 调用、6 小时分桶内容池和 60 条批量 prompt 构造；支持 `reloadAIConfig()` 热重载；`openai_model` 可配置多个模型（逗号/换行分隔），按配置优先级调用，首选不可用时才 fallback，各模型独立指数退避（30s→16min）
- **`ai_curation.go`** — AI 词条管理：按小时内容池、最近 100 条生成结果及 good/bad 长期样本池的 scribble 持久化、并发保护和样本抽样
- **`config.go`** — 配置管理：从 scribble DB 读写配置，首次启动从环境变量初始化，支持运行时更新；Bot Token 和 Admin ID 变更后通过 `restartBot()` 热重载，无需手动重启。新增 `WebDomain` 字段用于 Webhook 域名配置；`GetOpenAIModels()` 解析多模型列表
- **`api.go`** — Fiber 路由注册：公开端点（`/api/today`、`/api/cache`、`/api/templates`、`/api/entries`，`BrowserOnlyMiddleware` Sec-Fetch-Site 校验 + 速率限制 5 次/分钟/IP 双重防护）、认证端点（`/api/auth/login`）、管理端点（配置、日志、API Token、AI 结果标注和词条库）、用户统计端点（`/api/user/:id/stats`）、Webhook 端点（`/webhook`）、SPA fallback（`/*`）。使用 Fiber 中间件做 JWT 认证、API Token 认证、浏览器校验和速率限制
- **`apiext.go`** — API Token 管理与用户统计：Token CRUD（生成/删除/列表/验证）、`APITokenMiddleware` 中间件、用户统计引擎（全量扫描 history + 增量更新）、统计缓存（内存 + scribble DB `datas/user_stats`）。`expireUserStatsDaily()` 在每日零点清理过期统计，`updateUserStatsOnFortune()` 在算命成功后增量更新
- **`main.go`** — 应用入口：Fiber app 初始化、`go:embed` 嵌入前端静态文件、Telegram Bot 启动（Webhook 或 Long Polling 降级）、启动时加载 API Token、用户统计和 AI 词条库缓存
- **`logbuffer.go`** — 日志缓冲：内存环形缓冲区 + 文件持久化，支持通过 API 远程查看日志
- **`nominate.go`** — 词条提名与投票系统：赞成/反对、快速通过/否决、相似度查重（Jaro 算法）
- **`chats.go`** — Telegram 私聊状态机（IDLE → NOMINATE）、命令路由、管理员权限
- **`ingresssss.go`** — Ingress 事件词条（IFS/ISS 概率触发）
- **`specialDay.go`** — 特殊日期词条（硬编码）
- **`userstreak.go`** — 用户连续签到 streak 追踪
- **`tools/annualgen/`** — 离线年终总结生成工具（LLM + fallback）

## 行为准则

### Go 后端

- **包结构**：所有 `.go` 文件位于 `tgbot/` 目录，`package main`，单二进制部署
- **HTTP 框架**：**Fiber v3** (`github.com/gofiber/fiber/v3`)，路由注册在 `api.go` 的 `SetupRoutes()` 中
- **随机数**：**必须**使用 `crypto/rand`（安全性要求），禁止 `math/rand`
- **模板语法**：词条模板使用 `{{模板变量名}}` 格式，由 `fasttemplate` 解析
- **数据持久化**：通过 `scribble` 的 `db.Read()`/`db.Write()` 操作，数据文件位于 `db/datas/` 和 `db/history/`
- **并发模型**：后台任务使用 goroutine + `time.Ticker`（如 `laohuangli.update()`、`nomination.update()`、AI 内容刷新）
- **时区**：所有时间操作使用 `Asia/Shanghai`（通过 `time/tzdata` embed），Go 时间格式常量 `"2006-01-02 15:04"`
- **配置管理**：配置存储在 `db/datas/config.json`（scribble DB），首次启动从环境变量读取并初始化，之后优先从 DB 读取。支持通过 Web 管理面板运行时修改。Bot Token、Admin ID 和 WebDomain 变更后自动热重载（`restartBot()`），OpenAI 配置变更也自动热重载（`reloadAIConfig()`）。环境变量包括 `KUMA_PUSH_URL`、`ADMIN_USERNAME`、`ADMIN_PASSWORD`、`VALID_ANNUAL`、`TZ`、`WEB_DOMAIN`。`BOT_TOKEN`、`BOT_ADMIN_ID`、`OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL` 仅通过 Web 管理面板设置，不从环境变量读取
- **Telegram 连接**：优先使用 **Webhook** 模式（需配置 `WEB_DOMAIN`），通过 `b.ProcessUpdate()` 在 Fiber 路由中处理推送；未配置域名时降级为 Long Polling
- **JWT 认证**：使用 `golang-jwt/jwt/v5`，token 有效期 24 小时，通过 `Authorization: Bearer <token>` header 或 `auth_token` cookie 传递
- **日志系统**：`logbuffer.go` 实现内存环形缓冲（1000 条）+ 文件持久化（`db/datas/bot.log`），通过 `/api/admin/logs` API 远程查看
- **字符串相似度**：使用 `adrg/strutil` + `Jaro` 算法，阈值 0.9 为重复判定、0.5 为提示阈值
- **注释语言**：中文注释为主，函数/变量命名使用英文驼峰

### 前端

- **框架**：SvelteKit 2 + Svelte 4，使用 **SPA 模式**（`adapter-static` + `fallback: 'index.html'`）
- **样式**：TailwindCSS + DaisyUI 组件库
- **代码规范**：Prettier + ESLint（`npm run lint` / `npm run format`）
- **数据获取**：通过 `+page.js` 客户端 load 函数调用 Go 后端 API（`/api/cache`、`/api/templates`、`/api/entries`），API 路径为相对路径（同源）
- **管理后台**：`/login` 登录页 + `/admin` 管理路由组（配置编辑、日志查看、API Token、AI 结果标注和 good/bad 词条库管理），通过 localStorage 中的 JWT token 认证，客户端路由守卫（`+layout.js`）检查 token 有效性，未登录自动重定向；AI 模型支持多行输入（每行一个）
- **SSR**：全局禁用（`+layout.js` 中 `export const ssr = false`）

### 部署与构建

- **首次部署**：`cp .env-default .env` → 填写配置 → `make`
- **升级**：`git pull && make upgrade`
- **清理**：`make clean`
- **备份**：`make backup`（打包 `db/` 目录）
- **构建流程**：Dockerfile 多阶段构建（前端 npm build → Go embed + build → scratch 镜像）
- **反向代理**：Caddy 将 HTTPS 443 端口反代到容器 80 端口
- **Webhook**：配置 `WEB_DOMAIN` 后，Bot 启动时自动注册 Webhook（`https://<domain>/webhook`），Caddy 将请求转发到 Go Fiber

## 关键约束

- **词条长度**：单条提名不超过 **64 个 Unicode 字符**
- **提名限制**：每用户同时进行中的提名不超过 **5 条**
- **投票规则**：≥5 赞成票且赞成率 >66% 为通过；≥7 票且赞成率 >75% 为快速通过；≥5 票且反对多于赞成 为快速否决
- **AI 内容池**：每次从当前小时起生成连续 6 小时、每小时 10 条，共 60 条；当前和下一小时剩余总数少于 3 时异步追加同规格批次；仅发放当前小时词条，过期桶会清理
- **AI 多模型**：`openai_model` 支持多个模型（逗号/分号/换行分隔）；配置顺序即优先级，首选失败或退避时才依次 fallback；每个模型的重试退避单独计算（初始 30s，翻倍至上限 16min）
- **AI 质量样本**：最近 100 条已接受结果持久化；管理员可将词条互斥标记为 good/bad。生成时随机取最多 40 条 good 和 10 条 bad，数量不足时以默认样本补齐
- **今日缓存**：每日零点自动失效，缓存结果按用户 ID 存储，同一用户当天结果不变

## ⚠️ 自我进化约束（Critical Self-Sync Rule）

> **当 Agent 对项目架构、核心逻辑、依赖库或开发流程做出任何实质性修改或优化后，必须同步评估并更新本 `AGENTS.md` 中的相关条款，以确保该文档始终反映项目的最新真实状态。**
>
> 触发条件包括但不限于：
> - 新增/删除/重构关键模块
> - 升级或替换核心依赖
> - 修改投票规则、模板语法、AI 生成逻辑等业务规则
> - 变更部署方式、环境变量、端口配置
> - 引入新的编码规范或工具链
