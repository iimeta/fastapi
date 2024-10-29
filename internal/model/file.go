package model

import (
	"github.com/gogf/gf/v2/net/ghttp"
)

// Files接口请求参数
type FileFilesReq struct {
	Model    string            `json:"model" v:"required"`
	File     *ghttp.UploadFile `json:"file" type:"file" v:"required"`
	Purpose  string            `json:"purpose"`
	FilePath string            `json:"-"`
}
