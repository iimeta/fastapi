package model

import (
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

type ImageRes struct {
	Data         []smodel.ImageResponseData // 图像数据
	Error        error                      // 错误信息
	ConnTime     int64                      // 连接时间
	Duration     int64                      // 持续时间
	TotalTime    int64                      // 总时间
	InternalTime int64                      // 内耗时间
	EnterTime    int64                      // 进入时间
}
