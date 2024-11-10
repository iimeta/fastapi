#!/bin/bash

docker run --name fastapi -d -p 8000:8000 \
  --network host \
  --restart=always \
  -v /etc/localtime:/etc/localtime:ro \
  -v /data/fastapi:/app \
  iimeta/fastapi:latest
