<div align=center>

<img src="https://iim.ai/public/images/logo.png?t=2023112623" width="128"/>

# 智元 Fast API

[![Release](https://img.shields.io/github/v/release/iimeta/fastapi?color=blue)](https://github.com/iimeta/fastapi/releases)
[![License](https://img.shields.io/static/v1?label=license&message=MIT&color=green)](https://github.com/iimeta/fastapi/blob/main/LICENSE)
[![Docker Pull](https://img.shields.io/docker/pulls/iimeta/fastapi?color=brightgreen)](https://hub.docker.com/r/iimeta/fastapi)

</div>

`智元 Fast API` 是一站式API管理系统，将各类大模型API进行统一格式、统一规范、统一管理，使其在功能、性能和用户体验上达到极致。 

## 手动Docker部署
0. 需提前安装好Redis和MongoDB以及Docker服务
1. 拉取镜像: docker pull iimeta/fastapi:0.1.0
2. 创建配置文件目录: mkdir -p /data/fastapi/manifest/config
5. 进入配置文件目录: cd /data/fastapi/manifest/config
6. 下载配置文件: wget https://github.com/iimeta/fastapi/raw/docker/manifest/config/config.yaml
7. 修改配置文件: vi config.yaml 按 config.yaml 中配置项的注释, 修改对应参数
8. 启动服务: 
```shell
docker run -d \
  --network host \
  --restart=always \
  -p 8000:8000 \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi/manifest/config/config.yaml:/app/manifest/config/config.yaml \
  --name fastapi \
  iimeta/fastapi:0.1.0
```

## 自动Docker部署
0. 需提前安装好Redis和MongoDB以及Docker服务
1. 下载一键部署脚本: wget https://github.com/iimeta/fastapi/raw/docker/bin/install.sh
2. 执行安装: chmod +777 install.sh && ./install.sh
3. 修改配置文件: vi /data/fastapi/manifest/config/config.yaml 按 config.yaml 中配置项的注释, 修改对应参数
4. 启动服务: chmod +777 start.sh && ./start.sh

## 备用国内镜像地址
```shell
docker pull registry.cn-hangzhou.aliyuncs.com/iimeta/fastapi:0.1.0
```
```shell
docker run -d \
  --network host \
  --restart=always \
  -p 8000:8000 \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi/manifest/config/config.yaml:/app/manifest/config/config.yaml \
  --name fastapi \
  registry.cn-hangzhou.aliyuncs.com/iimeta/fastapi:0.1.0
```


![Star History Chart](https://api.star-history.com/svg?repos=iimeta/fastapi&type=Date)
