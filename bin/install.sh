#!/bin/bash

docker pull iimeta/fastapi:0.1.0

mkdir -p /data/fastapi/manifest/config

wget -P /data/fastapi/manifest/config https://github.com/iimeta/fastapi/raw/docker/manifest/config/config.yaml
wget https://github.com/iimeta/fastapi/raw/docker/bin/start.sh
