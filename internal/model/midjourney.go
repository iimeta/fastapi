package model

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
)

type MidjourneyResponse struct {
	smodel.MidjourneyResponse
	ReqUrl       string       `json:"req_url"`   // 请求地址
	TaskId       string       `json:"task_id"`   // 任务ID
	Action       string       `json:"action"`    // 动作[IMAGINE, UPSCALE, VARIATION, ZOOM, PAN, DESCRIBE, BLEND, SHORTEN, SWAP_FACE]
	Prompt       string       `json:"prompt"`    // 提示(提问)
	PromptEn     string       `json:"prompt_en"` // 英文提示(提问)
	ImageUrl     string       `json:"image_url"` // 图像地址
	Progress     string       `json:"progress"`  // 进度
	Usage        smodel.Usage `json:"usage"`
	Error        error        `json:"err"`
	ConnTime     int64        `json:"-"`
	Duration     int64        `json:"-"`
	TotalTime    int64        `json:"-"`
	InternalTime int64        `json:"-"`
	EnterTime    int64        `json:"-"`
}
