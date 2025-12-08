package model

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
)

type ImageRes struct {
	Data         []smodel.ImageResponseData // 图像数据
	Error        error                      // 错误信息
	TotalTime    int64                      // 总时间
	InternalTime int64                      // 内耗时间
	EnterTime    int64                      // 进入时间
}
