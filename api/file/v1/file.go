package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/model"
)

// Upload接口请求参数
type UploadReq struct {
	g.Meta   `path:"/" tags:"file" method:"post" summary:"Upload接口"`
	Provider string `json:"provider"`
	smodel.FileUploadRequest
}

// Upload接口响应参数
type UploadRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// List接口请求参数
type ListReq struct {
	g.Meta `path:"/" tags:"file" method:"get" summary:"List接口"`
	smodel.FileListRequest
}

// List接口响应参数
type ListRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Retrieve接口请求参数
type RetrieveReq struct {
	g.Meta `path:"/{file_id}" tags:"file" method:"get" summary:"Retrieve接口"`
	smodel.FileRetrieveRequest
}

// Retrieve接口响应参数
type RetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Delete接口请求参数
type DeleteReq struct {
	g.Meta `path:"/{file_id}" tags:"file" method:"delete" summary:"Delete接口"`
	smodel.FileDeleteRequest
}

// Delete接口响应参数
type DeleteRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Content接口请求参数
type ContentReq struct {
	g.Meta `path:"/{file_id}/content" tags:"file" method:"get" summary:"Content接口"`
	smodel.FileContentRequest
}

// Content接口响应参数
type ContentRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
