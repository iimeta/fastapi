<div align=center>

<img src="https://iim.ai/public/images/logo.png?t=20231207" width="128"/>

# 智元 Fast API

[![Release](https://img.shields.io/github/v/release/iimeta/fastapi?color=blue)](https://github.com/iimeta/fastapi/releases)
[![License](https://img.shields.io/static/v1?label=license&message=MIT&color=green)](https://github.com/iimeta/fastapi/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/iimeta/fastapi)](https://goreportcard.com/report/github.com/iimeta/fastapi)

</div>

`智元 Fast API` 是一站式API管理系统，将各类LLM API进行统一格式、统一规范、统一管理，使其在功能、性能和用户体验上达到极致。

## 支持模型

| 公司                                  | Completion | Image | Audio | Multimodal | Realtime | Embedding | Moderation |
|---------------------------------------|------------|-------|-------|------------|----------|-----------|------------|
| OpenAI<br/>Azure                      | ✔️         | ✔️    | ✔️    | ✔️         | ✔️       | ✔️       | ✔️         |
| 百度                                  | ✔️         |       |       |            |          |           |            |
| 科大讯飞                              | ✔️         | ✔️    |       |            |          |           |            |
| 阿里云                                | ✔️         |       |       |            |          |           |            |
| 智谱AI                                | ✔️         |       |       |            |          |           |            |
| Google                                | ✔️         |       |       | ✔️         |          |  ️         |  ️          |
| DeepSeek                              | ✔️         |       |       |            |          |           |            |
| 360智脑                               | ✔️         |       |       |            |          |           |            |
| Midjourney                            |            | ✔️    |       |            |          |           |            |
| Anthropic<br/>GCPClaude<br/>AWSClaude | ✔️         |       |       |            |          |           |            |

## 项目预览

- 用户端: https://demo.fastapi.ai/login
- 账号/密码: demo@fastapi.ai/123456

- 管理端: https://demo.fastapi.ai/admin
- 账号/密码: admin/admin123

## 部署支持

✔️ 集群部署  
✔️ 多地部署  
✔️ 跨区部署

## Docker部署
### [传送门](https://github.com/iimeta/fastapi/tree/docker)

### [部署视频](https://files.fastapi.ai/public/video/install.mp4)
https://files.fastapi.ai/public/video/install.mp4

## 公益站点

- API地址: https://api.free.fastapi.ai
- 注册后请联系作者领取1000万额度

## 项目截图
### 管理端
![dashboard](https://github.com/iimeta/fastapi-admin/assets/138393700/4d26474f-2082-41b4-8bd9-843ccb66d0a7)
![model](https://github.com/iimeta/fastapi-admin/assets/138393700/89d0721c-a72b-47f3-84e3-0080f5f1c8f9)
![key](https://github.com/iimeta/fastapi-admin/assets/138393700/f93922de-92e2-438e-8786-ae26981ec154)
![modelagent](https://github.com/iimeta/fastapi-admin/assets/138393700/f1e2c21f-d5ce-45e7-811d-7a5ed1a04138)
![user](https://github.com/iimeta/fastapi-admin/assets/138393700/1e812ed7-ff9f-46ab-847d-23a177bbedf0)
![chat](https://github.com/iimeta/fastapi-admin/assets/138393700/041a4324-eba8-42d4-8467-5964df6242f3)

### 用户端
![dashboard](https://github.com/iimeta/fastapi-admin/assets/138393700/94cd152c-b370-4cd9-b3f2-ca1dddf6821e)
![mymodel](https://github.com/iimeta/fastapi-admin/assets/138393700/cd8fec82-b2b8-4af4-b471-2c313e321d30)
![app](https://github.com/iimeta/fastapi-admin/assets/138393700/4dec9a5f-f399-4bfc-a3e3-bb50248a2c4c)
![appkey](https://github.com/iimeta/fastapi-admin/assets/138393700/662fe51d-9ed1-4672-b9db-f418b492b9a2)
![chat](https://github.com/iimeta/fastapi-admin/assets/138393700/5a92ab02-1319-436c-af2d-7d9ef0f49fe5)

## API接口

### 同OpenAI接口
```bash
curl --location 'https://api.fastapi.ai/v1/chat/completions' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer sk-FastAPI1YzE0kXf0zNb0ldX1nBLDm1Bh0SoSK0G0PzR1tNxW' \
--data '{
    "model": "gpt-3.5-turbo",
    "stream": true,
    "messages": [
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "hi"}
    ]
}'
```

## 开源地址
| 仓库 | API                                          | Web                                                  | Admin                                                    | SDK                                                  |
| ---- |----------------------------------------------|------------------------------------------------------|----------------------------------------------------------|------------------------------------------------------|
| 主库 | [fastapi](https://github.com/iimeta/fastapi) | [fastapi-web](https://github.com/iimeta/fastapi-web) | [fastapi-admin](https://github.com/iimeta/fastapi-admin) | [fastapi-sdk](https://github.com/iimeta/fastapi-sdk) |
| 码云 | [fastapi](https://gitee.com/iimeta/fastapi)  | [fastapi-web](https://gitee.com/iimeta/fastapi-web)  | [fastapi-admin](https://gitee.com/iimeta/fastapi-admin)  | [fastapi-sdk](https://gitee.com/iimeta/fastapi-sdk)  |

## 作者
<img src="https://iim.ai/public/images/Author.png?t=20231207" width="320" alt="作者"/>

## 交流群
<img src="https://iim.ai/public/images/WeChatGroup.jpg?t=20241120" width="320" alt="交流群"/>


![Star History Chart](https://api.star-history.com/svg?repos=iimeta/fastapi&type=Date)
