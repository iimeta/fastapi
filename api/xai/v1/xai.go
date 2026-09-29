package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// VideoCreate接口请求参数
type VideoCreateReq struct {
	g.Meta `path:"/" tags:"xai" method:"post" summary:"VideoCreate接口"`
	smodel.XAIVideoCreateReq
}

// VideoCreate接口响应参数
type VideoCreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoRetrieve接口请求参数
type VideoRetrieveReq struct {
	g.Meta `path:"/{request_id}" tags:"xai" method:"get" summary:"VideoRetrieve接口"`
	smodel.XAIVideoRetrieveReq
}

// VideoRetrieve接口响应参数
type VideoRetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
