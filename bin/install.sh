#!/bin/bash

docker pull iimeta/fastapi:latest

mkdir -p /data/fastapi/manifest/config
mkdir -p /data/fastapi/resource/audio
mkdir -p /data/fastapi/resource/file

wget -P /data/fastapi/manifest/config https://github.com/iimeta/fastapi/raw/docker/manifest/config/config.yaml
wget https://github.com/iimeta/fastapi/raw/docker/bin/start.sh
