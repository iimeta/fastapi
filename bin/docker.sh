#!/bin/bash
cd `dirname $0`
cd ../

docker build -f ./bin/Dockerfile -t iimeta/fastapi:0.1.2 .
