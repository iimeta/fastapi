package model

import (
	"github.com/gogf/gf/v2/net/ghttp"
)

type FileReq struct {
	Action      string         // 接口
	RequestData map[string]any // 请求数据
}

type FileRes struct {
	FileId       string         // 文件ID
	ResponseData map[string]any // 响应数据
	Error        error          // 错误信息
	TotalTime    int64          // 总时间
	InternalTime int64          // 内耗时间
	EnterTime    int64          // 进入时间
}

// Files接口请求参数
type FileFilesReq struct {
	Model    string            `json:"model" v:"required"`
	File     *ghttp.UploadFile `json:"file" type:"file" v:"required"`
	Purpose  string            `json:"purpose"`
	FilePath string            `json:"-"`
}
