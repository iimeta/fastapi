<div align=center>

<img src="https://www.fastapi.ai/logo.png" width="128"/>

# 智元 Fast API

[![Release](https://img.shields.io/github/v/release/iimeta/fastapi?color=blue)](https://github.com/iimeta/fastapi/releases)
[![License](https://img.shields.io/static/v1?label=license&message=MIT&color=green)](https://github.com/iimeta/fastapi/blob/main/LICENSE)
[![Docker Pull](https://img.shields.io/docker/pulls/iimeta/fastapi?color=brightgreen)](https://hub.docker.com/r/iimeta/fastapi)

</div>

`智元 Fast API` 是一站式API管理系统，将各类大模型API进行统一格式、统一规范、统一管理，使其在功能、性能和用户体验上达到极致。

## 手动Docker部署
0. 需提前安装好Redis和MongoDB以及Docker服务
1. 拉取镜像: `docker pull iimeta/fastapi:latest`
2. 创建配置文件目录: `mkdir -p /data/fastapi/manifest/config`
3. 进入配置文件目录: `cd /data/fastapi/manifest/config`
4. 下载配置文件: `wget https://github.com/iimeta/fastapi/raw/docker/manifest/config/config.yaml`
5. 修改配置文件: `vi config.yaml` 按 config.yaml 中配置项的注释, 修改对应参数
6. 启动服务: 
```shell
docker run --name fastapi -d -p 8000:8000 \
  --network host \
  --restart=always \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi:/app \
  iimeta/fastapi:latest
```
7. 访问: http://ip:8000/ 如: http://127.0.0.1:8000/

## 自动Docker部署
0. 需提前安装好Redis和MongoDB以及Docker服务
1. 下载一键部署脚本: `wget -O install.sh https://github.com/iimeta/fastapi/raw/docker/bin/install.sh && sh install.sh`
2. 修改配置文件: `vi /data/fastapi/manifest/config/config.yaml` 按 config.yaml 中配置项的注释, 修改对应参数
3. 启动服务: `chmod +777 start.sh && ./start.sh`
4. 访问: http://ip:8000/ 如: http://127.0.0.1:8000/

## 备用国内镜像地址
```shell
docker pull registry.cn-hangzhou.aliyuncs.com/iimeta/fastapi:latest
```
```shell
docker run --name fastapi -d -p 8000:8000 \
  --network host \
  --restart=always \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi:/app \
  registry.cn-hangzhou.aliyuncs.com/iimeta/fastapi:latest
```


![Star History Chart](https://api.star-history.com/svg?repos=iimeta/fastapi&type=Date)
