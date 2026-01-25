<div align=center>

<img src="https://www.fastapi.ai/logo.png" width="128"/>

# 智元 Fast API

[![Release](https://img.shields.io/github/v/release/iimeta/fastapi?color=blue)](https://github.com/iimeta/fastapi/releases)
[![License](https://img.shields.io/static/v1?label=license&message=Apache-2.0&color=green)](https://github.com/iimeta/fastapi/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/iimeta/fastapi)](https://goreportcard.com/report/github.com/iimeta/fastapi)

</div>

企业级 LLM API 快速集成系统，支持DeepSeek、OpenAI、Azure、文心一言、讯飞星火、通义千问、智谱GLM、豆包、Gemini、Anthropic Claude以及OpenAI格式的模型等，简洁的页面风格，轻量高效且稳定，支持Docker一键部署。业务系统只需要按照统一API标准，对接一次的开发工作量，即可无缝对接N个大模型，无需考虑N个大模型背后的各种复杂逻辑等等，可大大降低开发和维护成本...

## 支持模型提供商

<table>
  <tr align="center">
    <td>
      <img src="https://www.fastapi.ai/assets/OpenAI.C7t6TQyq.ico" width="50" height="50" alt="OpenAI"><br/>
      <a href="https://openai.com/">OpenAI</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/Azure.DKMxtfNp.ico" width="50" height="50" alt="Azure"><br/>
      <a href="https://azure.microsoft.com/">Azure</a>
    </td>
	<td>
      <img src="https://chat.deepseek.com/favicon.svg" width="50" height="50" alt="DeepSeek"><br/>
      <a href="https://www.deepseek.com/">DeepSeek</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/Qwen.BdOnZtF3.png" width="50" height="50" alt="通义千问"><br/>
      <a href="https://tongyi.aliyun.com/">通义千问</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/Gemini.D7sh0nXm.png" width="50" height="50" alt="Gemini"><br/>
      <a href="https://gemini.google.com/">Gemini</a>
    </td>
  </tr>
  <tr align="center">
    <td>
      <img src="https://eb-static.cdn.bcebos.com/logo/favicon.ico" width="50" height="50" alt="文心一言"><br/>
      <a href="https://yiyan.baidu.com/">文心一言</a>
    </td>
    <td>
      <img src="https://chatglm.cn/img/icons/favicon.svg" width="50" height="50" alt="智谱清言"><br/>
      <a href="https://chatglm.cn/">智谱清言</a>
    </td>
	<td>
      <img src="https://portal.volccdn.com/obj/volcfe/misc/favicon.png" width="50" height="50" alt="火山引擎"><br/>
      <a href="https://www.volcengine.com/">火山引擎</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/Doubao.C5HENFx3.png" width="50" height="50" alt="豆包"><br/>
      <a href="https://www.doubao.com/">豆包</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/MiniMax.BJYeVzwi.ico" width="50" height="50" alt="MiniMax"><br/>
      <a href="https://www.minimaxi.com/">MiniMax</a>
    </td>
  </tr>
  <tr align="center">
    <td>
      <img src="https://www.fastapi.ai/assets/Claude.BAdojD7P.ico" width="50" height="50" alt="Anthropic"><br/>
      <a href="https://www.anthropic.com/">Anthropic</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/SparkDesk.BnGrALWW.ico" width="50" height="50" alt="讯飞星火"><br/>
      <a href="https://xinghuo.xfyun.cn/">讯飞星火</a>
    </td>
	<td>
      <img src="https://statics.moonshot.cn/kimi-web-seo/favicon.ico" width="50" height="50" alt="Kimi"><br/>
      <a href="https://kimi.moonshot.cn/">Kimi</a>
    </td>
    <td>
      <img src="https://www.fastapi.ai/assets/Grok.CIEyUb2Y.png" width="50" height="50" alt="Grok"><br/>
      <a href="https://x.ai/">Grok</a>
    </td>
    <td>
      <img src="https://siliconflow.cn/favicon.ico" width="50" height="50" alt="硅基流动"><br/>
      <a href="https://siliconflow.cn/">硅基流动</a>
    </td>
  </tr>
</table>

## API接口

### 同OpenAI接口

```bash
curl --location 'https://api.fastapi.ai/v1/chat/completions' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer sk-FastAPI1YzE0kXf0zNb0ldX1nBLDm1Bh0SoSK0G0PzR1tNxW' \
--data '{
    "model": "gpt-5",
    "stream": true,
    "messages": [
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "hi"}
    ]
}'
```

## 社区版预览

- 管理端: <https://demo.fastapi.ai/admin>
- 用户端: <https://demo.fastapi.ai/login>
- 代理商: <https://demo.fastapi.ai/reseller>
- 账号/密码均是: demo/123456

## 专业版预览

- 管理端: <https://demo.fastapi.pro/admin>
- 用户端: <https://demo.fastapi.pro/login>
- 代理商: <https://demo.fastapi.pro/reseller>
- 账号/密码均是: demo/123456

## 部署支持

✔️ [Docker部署](https://www.fastapi.ai/guide/#docker%E9%83%A8%E7%BD%B2%E6%95%99%E7%A8%8B)  
✔️ 集群部署  
✔️ 多地部署  

## 部署教程

### [快速开始](https://www.fastapi.ai/guide/)

## 公益站点

- 用户端: <https://free.fastapi.pro/login>
- 代理商: <https://free.fastapi.pro/reseller>
- API接口: <https://api.free.fastapi.pro>
- 注册就送 **$1,000,000** 额度, 支持注册代理商

## 开源地址

| 仓库 | API                                          | Web                                                  | Admin                                                    | SDK                                                  |
| ---- |----------------------------------------------|------------------------------------------------------|----------------------------------------------------------|------------------------------------------------------|
| 主库 | [fastapi](https://github.com/iimeta/fastapi) | [fastapi-web](https://github.com/iimeta/fastapi-web) | [fastapi-admin](https://github.com/iimeta/fastapi-admin) | [fastapi-sdk](https://github.com/iimeta/fastapi-sdk) |
| 码云 | [fastapi](https://gitee.com/iimeta/fastapi)  | [fastapi-web](https://gitee.com/iimeta/fastapi-web)  | [fastapi-admin](https://gitee.com/iimeta/fastapi-admin)  | [fastapi-sdk](https://gitee.com/iimeta/fastapi-sdk)  |

## 作者 / 交流群

<center>
  <img src="https://iim.ai/public/images/Author.png?t=20231207" width="320" alt="作者"/>
  <img src="https://iim.ai/public/images/WeChatGroup.jpg?t=20260125" width="320" alt="交流群"/>
</center>

![Star History Chart](https://api.star-history.com/svg?repos=iimeta/fastapi&type=Date)
