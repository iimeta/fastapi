package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/model"
)

// Files接口请求参数
type FilesReq struct {
	g.Meta `path:"/files" tags:"file" method:"post" summary:"Files接口"`
	model.FileFilesReq
}

// Files接口响应参数
type FilesRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
