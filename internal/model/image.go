package model

import (
	"github.com/gogf/gf/v2/net/ghttp"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model/common"
)

type ImageReq struct {
	Prompt         string `json:"prompt,omitempty"`
	Model          string `json:"model,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}

type ImageRes struct {
	Created         int64                           `json:"created,omitempty"`
	Data            []smodel.ImageResponseDataInner `json:"data,omitempty"`
	Usage           smodel.Usage                    `json:"usage"`
	Error           error                           `json:"err"`
	TotalTime       int64                           `json:"-"`
	InternalTime    int64                           `json:"-"`
	EnterTime       int64                           `json:"-"`
	GenerationQuota common.GenerationQuota          `json:"-"` // 生成额度
}

type ImageEditRequest struct {
	Image          []*ghttp.UploadFile `json:"image,omitempty"`
	Prompt         string              `json:"prompt,omitempty"`
	Background     string              `json:"background,omitempty"`
	Mask           *ghttp.UploadFile   `json:"mask,omitempty"`
	Model          string              `json:"model,omitempty"`
	N              int                 `json:"n,omitempty"`
	Quality        string              `json:"quality,omitempty"`
	ResponseFormat string              `json:"response_format,omitempty"`
	Size           string              `json:"size,omitempty"`
	User           string              `json:"user,omitempty"`
}
