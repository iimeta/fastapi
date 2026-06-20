package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// Generations接口请求参数
type GenerationsReq struct {
	g.Meta `path:"/generations" tags:"image" method:"post" summary:"Generations接口"`
	Async  bool `json:"async,omitempty"`
	smodel.ImageGenerationRequest
}

// Generations接口响应参数
type GenerationsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Edits接口请求参数
type EditsReq struct {
	g.Meta `path:"/edits" tags:"image" method:"post" summary:"Edits接口"`
	Async  bool `json:"async,omitempty"`
	smodel.ImageEditRequest
}

// Edits接口响应参数
type EditsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// List接口请求参数
type ListReq struct {
	g.Meta `path:"/" tags:"image" method:"get" summary:"List接口"`
	smodel.ImageListRequest
}

// List接口响应参数
type ListRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Retrieve接口请求参数
type RetrieveReq struct {
	g.Meta `path:"/{image_id}" tags:"image" method:"get" summary:"Retrieve接口"`
	smodel.ImageRetrieveRequest
}

// Retrieve接口响应参数
type RetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Delete接口请求参数
type DeleteReq struct {
	g.Meta `path:"/{image_id}" tags:"image" method:"delete" summary:"Delete接口"`
	smodel.ImageDeleteRequest
}

// Delete接口响应参数
type DeleteRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Content接口请求参数
type ContentReq struct {
	g.Meta `path:"/{image_id}/content" tags:"image" method:"get" summary:"Content接口"`
	smodel.ImageContentRequest
}

// Content接口响应参数
type ContentRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
