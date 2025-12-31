package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// Create接口请求参数
type CreateReq struct {
	g.Meta `path:"/" tags:"video" method:"post" summary:"Create接口"`
	smodel.VideoCreateRequest
}

// Create接口响应参数
type CreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Remix接口请求参数
type RemixReq struct {
	g.Meta `path:"/{video_id}/remix" tags:"video" method:"post" summary:"Remix接口"`
	smodel.VideoRemixRequest
}

// Remix接口响应参数
type RemixRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// List接口请求参数
type ListReq struct {
	g.Meta `path:"/" tags:"video" method:"get" summary:"List接口"`
	smodel.VideoListRequest
}

// List接口响应参数
type ListRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Retrieve接口请求参数
type RetrieveReq struct {
	g.Meta `path:"/{video_id}" tags:"video" method:"get" summary:"Retrieve接口"`
	smodel.VideoRetrieveRequest
}

// Retrieve接口响应参数
type RetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Delete接口请求参数
type DeleteReq struct {
	g.Meta `path:"/{video_id}" tags:"video" method:"delete" summary:"Delete接口"`
	smodel.VideoDeleteRequest
}

// Delete接口响应参数
type DeleteRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Content接口请求参数
type ContentReq struct {
	g.Meta `path:"/{video_id}/content" tags:"video" method:"get" summary:"Content接口"`
	smodel.VideoContentRequest
}

// Content接口响应参数
type ContentRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
