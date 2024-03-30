#!/bin/bash

docker run -d \
  --network host \
  --restart=always \
  -p 8000:8000 \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi/manifest/config/config.yaml:/app/manifest/config/config.yaml \
  --name fastapi \
  iimeta/fastapi:0.1.0
