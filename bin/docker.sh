#!/bin/bash
cd `dirname $0`
cd ../

docker build -f ./bin/Dockerfile -t iimeta/fastapi:2.1.0 .
