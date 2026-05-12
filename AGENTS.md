# AGENTS.md — Ingress老黄历 项目指南

## 项目 DNA

**Ingress老黄历** 是一个 Ingress 游戏主题的"每日算命" Telegram Bot + 配套网站，由老黄历Go分叉而来。

### 架构概览

```
├── tgbot/          # Go 后端 — Telegram Bot + API 服务 (package main)
├── website/        # SvelteKit 前端 — 展示页面 (Node adapter, 端口 4090)
├── db/             # scribble JSON 文件数据库 (运行时生成)
│   ├── datas/      # 词条库、模板、提名、缓存、streak
│   └── history/    # 每日历史快照
├── docker-compose.yml  # 双容器编排: tgbot + website
└── Makefile        # docker compose 快捷命令
```

### 核心技术栈

| 层 | 技术 |
|---|---|
| **Bot 后端** | **Go 1.24** · `telebot.v3` · `golang-scribble` (flat-file JSON DB) · `fasttemplate` (`{{var}}` 模板引擎) · `go-openai` · `crypto/rand` |
| **前端** | **SvelteKit 2** · Svelte 4 · **TailwindCSS + DaisyUI** · `@sveltejs/adapter-node` |
| **部署** | **Docker Compose** · `.env` 配置 · Makefile 管理生命周期 |

### 关键模块

- **`laohuangli.go`** — 核心引擎：词条库加载、加权均衡随机抽取、模板渲染、每日缓存、今日指引生成
- **`artificialIdiot.go`** — AI 内容生成：OpenAI API 调用、内容池管理（整点切换 + 预取）、prompt 构造；支持 `reloadAIConfig()` 热重载
- **`config.go`** — 配置管理：从 scribble DB 读写配置，首次启动从环境变量初始化，支持运行时更新；Bot Token 和 Admin ID 变更后通过 `restartBot()` 热重载，无需手动重启
- **`api.go`** — HTTP API 服务：公开端点（`/api/cache`、`/api/templates`、`/api/entries`）、认证端点（`/api/auth/login`）、管理端点（`/api/admin/config`、`/api/admin/logs`），JWT 认证中间件
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
- **随机数**：**必须**使用 `crypto/rand`（安全性要求），禁止 `math/rand`
- **模板语法**：词条模板使用 `{{模板变量名}}` 格式，由 `fasttemplate` 解析
- **数据持久化**：通过 `scribble` 的 `db.Read()`/`db.Write()` 操作，数据文件位于 `db/datas/` 和 `db/history/`
- **并发模型**：后台任务使用 goroutine + `time.Ticker`（如 `laohuangli.update()`、`nomination.update()`、AI 内容刷新）
- **时区**：所有时间操作使用 `Asia/Shanghai`（通过 `time/tzdata` embed），Go 时间格式常量 `"2006-01-02 15:04"`
- **配置管理**：配置存储在 `db/datas/config.json`（scribble DB），首次启动从环境变量读取并初始化，之后优先从 DB 读取。支持通过 Web 管理面板运行时修改。Bot Token 和 Admin ID 变更后自动热重载（`restartBot()`），OpenAI 配置变更也自动热重载（`reloadAIConfig()`）。环境变量包括 `KUMA_PUSH_URL`、`ADMIN_USERNAME`、`ADMIN_PASSWORD`、`VALID_ANNUAL`、`TZ`。`BOT_TOKEN`、`BOT_ADMIN_ID`、`OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL` 仅通过 Web 管理面板设置，不从环境变量读取
- **JWT 认证**：使用 `golang-jwt/jwt/v5`，token 有效期 24 小时，通过 `Authorization: Bearer <token>` header 或 `auth_token` cookie 传递
- **日志系统**：`logbuffer.go` 实现内存环形缓冲（1000 条）+ 文件持久化（`db/datas/bot.log`），通过 `/api/admin/logs` API 远程查看
- **字符串相似度**：使用 `adrg/strutil` + `Jaro` 算法，阈值 0.9 为重复判定、0.5 为提示阈值
- **注释语言**：中文注释为主，函数/变量命名使用英文驼峰

### 前端

- **框架**：SvelteKit 2 + Svelte 4，使用 SSG/Node adapter 模式
- **样式**：TailwindCSS + DaisyUI 组件库
- **代码规范**：Prettier + ESLint（`npm run lint` / `npm run format`）
- **数据获取**：通过 `+page.server.js` 调用 Go 后端 API 获取数据（`/api/cache`、`/api/templates`、`/api/entries`）
- **管理后台**：`/login` 登录页 + `/admin` 管理路由组（配置编辑、日志查看），通过 JWT cookie 认证，未登录自动重定向

### 部署与构建

- **首次部署**：`cp .env-default .env` → 填写配置 → `make`
- **升级**：`git pull && make upgrade`
- **清理**：`make clean`
- **备份**：`make backup`（打包 `db/` 目录）

## 关键约束

- **词条长度**：单条提名不超过 **64 个 Unicode 字符**
- **提名限制**：每用户同时进行中的提名不超过 **5 条**
- **投票规则**：≥5 赞成票且赞成率 >66% 为通过；≥7 票且赞成率 >75% 为快速通过；≥5 票且反对多于赞成 为快速否决
- **AI 内容池**：每小时刷新，池上限 20 条，常规模式池 <5 时触发更新，59 分进入预取模式
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
