package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// Create接口请求参数
type CreateReq struct {
	g.Meta   `path:"/" tags:"batch" method:"post" summary:"Create接口"`
	Provider string `json:"provider"`
	smodel.BatchCreateRequest
}

// Create接口响应参数
type CreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// List接口请求参数
type ListReq struct {
	g.Meta `path:"/" tags:"batch" method:"get" summary:"List接口"`
	smodel.BatchListRequest
}

// List接口响应参数
type ListRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Retrieve接口请求参数
type RetrieveReq struct {
	g.Meta `path:"/{batch_id}" tags:"batch" method:"get" summary:"Retrieve接口"`
	smodel.BatchRetrieveRequest
}

// Retrieve接口响应参数
type RetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Cancel接口请求参数
type CancelReq struct {
	g.Meta `path:"/{batch_id}/cancel" tags:"batch" method:"post" summary:"Cancel接口"`
	smodel.BatchCancelRequest
}

// Cancel接口响应参数
type CancelRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
