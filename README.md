<div align="center">
<img src="https://socialify.git.ci/szres/ingress-laohuangli/image?font=KoHo&language=1&logo=https%3A%2F%2Fraw.githubusercontent.com%2Fszres%2Fingress-laohuangli%2Fmain%2Fwebsite%2Fstatic%2Ffavicon.png&name=1&pattern=Circuit%20Board&stargazers=1&theme=Auto" alt="ingress-laohuangli" width="640" height="320" />
</div>
   
# Ingress老黄历

[![Chat on Telegram](https://img.shields.io/badge/@ingress_laohuangli_bot-2CA5E0.svg?logo=telegram&label=Telegram)](https://t.me/ingress_laohuangli_bot)
![GitHub Repo stars](https://img.shields.io/github/stars/szres/ingress-laohuangli?style=flat&color=ffaaaa)
[![Software License](https://img.shields.io/github/license/szres/ingress-laohuangli)](LICENSE)
![Docker](https://img.shields.io/badge/Build_with-Docker-ffaaaa)

这个项目是对 [老黄历Go](https://github.com/szres/laohuangli-lite-go) 项目的Ingress主题分叉。增加了Ingress活动相关的定期词条，并且增加了OpenAI生成结果的支持。

## 部署

> tips: 要使得 bot 正常工作需要在 `bot father` 处打开 bot 的 `inline` 功能

首先拷贝 `.env-default` 为 `.env`

1. 在 `.env` 中设置必要信息
   - `ADMIN_USERNAME`: 管理后台用户名（默认 `admin`）
   - `ADMIN_PASSWORD`: 管理后台密码（默认 `laohuangli`）
   - `KUMA_PUSH_URL`: 使用 [kuma-push](https://github.com/Nigh/kuma-push) 驱动的 [uptime-Kuma](https://github.com/louislam/uptime-kuma "uptimeKuma") 监控服务的推送地址，不带参数 **[可留空]**
   - `VALID_ANNUAL`: 年终总结展示年份（例如 2024） **[可留空]**

2. 启动服务后，通过 `http://your-domain:4090/admin` 登录管理后台，在 Web 界面中设置以下配置：
   - `BOT_TOKEN`: Telegram的bot token **[必填项]**
   - `BOT_ADMIN_ID`: 机器人管理员的Telegram ID **[可留空]**
   - `OPENAI_API_KEY`: OPENAI的api key **[可留空]**
   - `OPENAI_BASE_URL`: 自定义 OpenAI API 地址 **[可留空]**
   - `OPENAI_MODEL`: 自定义模型名称 **[可留空]**

   > **注意**：配置存储在数据库中，通过 Web 管理面板修改后即时生效（AI 配置自动热重载），无需重启服务。

3. 根据需要运行下面的命令

```shell
# 初次运行
make
# 拉取源码升级
git pull
make upgrade
# 移除容器
make clean
```

4. `website` 容器包含了一个 `node` 驱动的前端页面，前端页面默认暴露于 `4090` 端口，功能包括：
   - **首页**：展示当日算命信息
   - **提名助手**：查看模板词条信息，方便用户提名含有模板的词条
   - **管理后台**：通过 `http://your-domain:4090/admin` 访问（需登录），可查看实时日志、在线修改 Bot Token、管理员 ID、AI 配置等，无需重启服务

## API 接口

服务提供以下 RESTful API，所有路径相对于服务根地址（如 `http://localhost:4090`）。

### 公开接口

浏览器安全校验：所有公开接口通过 `Sec-Fetch-Site` 请求头校验仅允许浏览器访问（同源/同站请求），非浏览器请求返回 `403 Forbidden`。

速率限制：每 IP 每分钟最多 5 次请求，超限返回 `429 Too Many Requests`。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/today` | 获取今日已算命用户数量（返回 `{date, count}`） |
| `GET` | `/api/cache` | 获取今日缓存（返回 `{date, today, caches}`，包含今日指引和众生列表） |
| `GET` | `/api/templates` | 获取模板列表（返回模板 map） |
| `GET` | `/api/entries` | 获取词条列表（返回 `{entries, entries_user}`） |

`/api/cache` 响应示例：
```json
{
  "date": "2026-05-27",
  "today": {
    "clothing": { "positive": "...", "negative": "..." },
    "food": { "positive": "...", "negative": "..." },
    "travel": { "positive": "...", "negative": "..." }
  },
  "caches": {
    "123456": { "name": "AgentName", "result": "..." }
  }
}
```

### 认证接口

| 方法 | 路径 | 认证方式 | 说明 |
|------|------|----------|------|
| `POST` | `/api/auth/login` | 无（请求体 `{username, password}`） | 管理员登录，返回 JWT token |

### 管理接口（需 JWT 认证）

请求头：`Authorization: Bearer <jwt_token>`

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/admin/config` | 获取配置（敏感字段脱敏） |
| `PUT` | `/api/admin/config` | 更新配置（部分更新，仅传需修改的字段） |
| `GET` | `/api/admin/logs` | 获取运行日志（可选参数 `?n=200` 指定条数） |
| `GET` | `/api/admin/tokens` | 列出所有 API Token（token 脱敏显示） |
| `POST` | `/api/admin/tokens` | 创建新 API Token（请求体 `{name}`） |
| `DELETE` | `/api/admin/tokens/:id` | 删除指定 API Token |

### 用户统计接口（需 API Token 认证）

请求头：`Authorization: Bearer <api_token>`（通过管理后台「API Token」页面生成）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/user/:id/stats` | 获取用户算命统计（`:id` 为 Telegram 用户 ID） |

响应示例：
```json
{
  "user_id": "123456",
  "total_count": 150,
  "streak_count": 21,
  "current_streak": 5,
  "last_month_count": 25,
  "last_fortune_date": "2026-05-27"
}
```

### 接口测试

```shell
# 1. 获取今日已算命人数（公开接口，需浏览器 Sec-Fetch-Site 头，每分钟限 5 次）
curl -H 'Sec-Fetch-Site: same-origin' http://localhost:4090/api/today

# 2. 获取今日缓存（公开接口，需浏览器头）
curl -H 'Sec-Fetch-Site: same-origin' http://localhost:4090/api/cache

# 3. 获取模板列表（公开接口，需浏览器头）
curl -H 'Sec-Fetch-Site: same-origin' http://localhost:4090/api/templates

# 4. 获取词条列表（公开接口，需浏览器头）
curl -H 'Sec-Fetch-Site: same-origin' http://localhost:4090/api/entries

# 5. 无浏览器头请求 → 403 Forbidden
curl http://localhost:4090/api/cache

# 6. 管理员登录获取 JWT
TOKEN=$(curl -s -X POST http://localhost:4090/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"laohuangli"}' | jq -r '.token')

# 7. 查看配置
curl -H "Authorization: Bearer $TOKEN" http://localhost:4090/api/admin/config

# 8. 创建 API Token
API_TOKEN=$(curl -s -X POST http://localhost:4090/api/admin/tokens \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"测试 Token"}' | jq -r '.token')
echo "API Token: $API_TOKEN"

# 9. 列出所有 API Token
curl -H "Authorization: Bearer $TOKEN" http://localhost:4090/api/admin/tokens

# 10. 使用 API Token 查询用户统计（替换 USER_ID 为实际 Telegram 用户 ID）
curl -H "Authorization: Bearer $API_TOKEN" http://localhost:4090/api/user/123456/stats

# 11. 删除 API Token（替换 TOKEN_ID 为实际 Token ID）
curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:4090/api/admin/tokens/TOKEN_ID

# 12. 测试速率限制（连续请求 6 次，第 6 次应返回 429）
for i in $(seq 1 6); do curl -s -o /dev/null -w "请求 $i: HTTP %{http_code}\n" -H 'Sec-Fetch-Site: same-origin' http://localhost:4090/api/today; done
```

## 数据

词条与历史均使用[scribble](https://github.com/nanobox-io/golang-scribble)数据库保存。存放在项目根目录下。目录结构如下：

```
db/
├── datas/
│   ├── config.json           #应用配置（从 .env 初始化，可通过 web 管理面板修改）
│   ├── laohuangli-user.json  #用户提名词条
│   ├── laohuangli.json       #本地词条
│   ├── templates.json        #词条模板
│   ├── api_tokens.json       #API Token 存储
│   ├── user_stats.json       #用户统计缓存
│   └── bot.log               #运行日志
└── history/
    └── $date.json            #历史记录
```

## 年终总结生成

使用离线工具生成年度总结与 fallback 文案（需要设置 `OPENAI_API_KEY`）：

```shell
# 载入 .env 环境变量
set -a
source .env
set +a

# 生成 fallback 文案（建议先做一次）
cd tgbot
go run ./tools/annualgen -year=2024 -generate-fallback -db ../db -fallback ../db/annual/fallback.json

# 生成年度总结（默认读取 fallback.json，算命次数>30 用 LLM，总结候选语句上限30）
go run ./tools/annualgen -year=2024 -db ../db -fallback ../db/annual/fallback.json
```

可选参数：

- `-threshold=30`：进入 LLM 总结的最小次数
- `-candidates=30`：每人候选语句数量上限
- `-out=路径`：指定输出文件
- `-dry-run`：只统计不写文件

#### 词条结构

```json
{
  "uuid": "唯一ID",
  "content": "词条内容",
  "nominator": "提名人昵称"
}
```

#### 模板结构

```json
"模板变量名": {
  "desc": "模板描述",
  "values": [
    "模板内容数组1",
    "模板内容数组2",
    "模板内容数组3",
    "......"
  ]
}
```

### 词条示例

templates.json

```json
{
  "haircolor": {
  "desc": "适用于发色的单字颜色",
  "values": [
  	"红",
  	"粉",
  	"黄",
  	"蓝",
  	"绿",
  	"白",
  	"金"
  ]
  }
}
```

laohuangli-user.json

```json
{
  "uuid": "1",
  "content": "给老黄历提名新词条",
  "nominator": "匿名"
},
{
  "uuid": "2",
  "content": "染成{{haircolor}}毛",
  "nominator": "倪明"
}
```
