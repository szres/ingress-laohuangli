<div align="center">
<img src="https://socialify.git.ci/szres/ingress-laohuangli/image?font=KoHo&language=1&logo=https%3A%2F%2Fraw.githubusercontent.com%2Fszres%2Fingress-laohuangli%2Fmain%2Fwebsite%2Fstatic%2Ffavicon.png&name=1&pattern=Circuit%20Board&stargazers=1&theme=Auto" alt="laohuangli-lite-go" width="640" height="320" />
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
   - `BOT_TOKEN`: Telegram的bot token **[必填项]**
   - `OPENAI_API_KEY`: OPENAI的api key **[可留空]**
   - `BOT_ADMIN_ID`: 机器人管理员的Telegram ID，配置为管理员的ID可以使用更多命令 **[可留空]**
   - `KUMA_PUSH_URL`: 使用 [kuma-push](https://github.com/Nigh/kuma-push) 驱动的 [uptime-Kuma](https://github.com/louislam/uptime-kuma "uptimeKuma") 监控服务的推送地址，不带参数 **[可留空]**
   - `WEB_DOMAIN`: 老黄历网页的托管地址 **[可留空]**
2. 根据需要运行下面的命令

```shell
# 初次运行
make
# 拉取源码升级
git pull
make upgrade
# 移除容器
make clean
```

3. `website` 容器包含了一个 `node` 驱动的前端页面用于展示当日算命信息与查看模板词条信息，方便用户提名含有模板的词条。前端页面默认暴露于 `4090` 端口。

## 数据

词条与历史均使用[scribble](https://github.com/nanobox-io/golang-scribble)数据库保存。存放在项目根目录下。目录结构如下：

```
db/
├── datas/
│   ├── laohuangli-user.json  #用户提名词条
│   ├── laohuangli.json       #本地词条
│   └── templates.json        #词条模板
└── history/
    └── $date.json            #历史记录
```

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
