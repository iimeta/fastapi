package model

import (
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

type MidjourneyProxyResponse struct {
	sdkm.MidjourneyProxyResponse
	Usage        sdkm.Usage `json:"usage"`
	Error        error      `json:"err"`
	ConnTime     int64      `json:"-"`
	Duration     int64      `json:"-"`
	TotalTime    int64      `json:"-"`
	InternalTime int64      `json:"-"`
	EnterTime    int64      `json:"-"`
}
