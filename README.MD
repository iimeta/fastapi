<div align=center>

<img src="https://iim.ai/public/images/logo.png?t=20231207" width="128"/>

# 智元 Fast API

[![Release](https://img.shields.io/github/v/release/iimeta/fastapi?color=blue)](https://github.com/iimeta/fastapi/releases)
[![License](https://img.shields.io/static/v1?label=license&message=MIT&color=green)](https://github.com/iimeta/fastapi/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/iimeta/fastapi)](https://goreportcard.com/report/github.com/iimeta/fastapi)

</div>

`智元 Fast API` 是一站式API管理系统，将各类大模型API进行统一格式、统一规范、统一管理，使其在功能、性能和用户体验上达到极致。

## 功能介绍
- 支持 OpenAI 全模型

## 项目预览

- 用户端: https://demo.fastapi.ai/login
- 账号/密码: demo@fastapi.ai/123456

- 管理端: https://demo.fastapi.ai/admin
- 账号/密码: admin/admin123

### 部署视频
https://www.fastapi.ai/public/video/install.mp4

## Docker部署
### [传送门](https://github.com/iimeta/fastapi/tree/docker)

## API接口

### 同OpenAI接口
```bash
curl --location 'https://api.fastapi.ai/v1/chat/completions' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer sk-FastAPI1YzE0kXf0zNb0ldX1nBLDm1Bh0SoSK0G0PzR1tNxW' \
--data '{
    "model": "gpt-3.5-turbo-16k",
    "stream": true,
    "messages": [
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "hi"}
    ]
}'
```

## 开源地址
| 仓库 | 前端                                  | 后端                                    | API                               |
| ---- | ------------------------------------- | --------------------------------------- | ----------------------------------|
| 主库 | https://github.com/iimeta/fastapi-web | https://github.com/iimeta/fastapi-admin | https://github.com/iimeta/fastapi |
| 码云 | https://gitee.com/iimeta/fastapi-web  | https://gitee.com/iimeta/fastapi-admin  | https://gitee.com/iimeta/fastapi  |

## 作者
<img src="https://iim.ai/public/images/Author.png?t=20231207" width="320" alt="作者"/>


![Star History Chart](https://api.star-history.com/svg?repos=iimeta/fastapi&type=Date)
