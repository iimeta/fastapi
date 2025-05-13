package model

import (
	"github.com/gogf/gf/v2/net/ghttp"
	sdkm "github.com/iimeta/fastapi-sdk/model"
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
	Created      int64                         `json:"created,omitempty"`
	Data         []sdkm.ImageResponseDataInner `json:"data,omitempty"`
	Usage        sdkm.Usage                    `json:"usage"`
	Error        error                         `json:"err"`
	TotalTime    int64                         `json:"-"`
	InternalTime int64                         `json:"-"`
	EnterTime    int64                         `json:"-"`
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
