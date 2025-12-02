package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/model"
)

// Create接口请求参数
type CreateReq struct {
	g.Meta `path:"/" tags:"video" method:"post" summary:"Create接口"`
	smodel.VideoRequest
}

// Create接口响应参数
type CreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Remix接口请求参数
type RemixReq struct {
	g.Meta  `path:"/{video_id}/remix" tags:"video" method:"post" summary:"Remix接口"`
	VideoId string `json:"video_id"`
}

// Remix接口响应参数
type RemixRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// List接口请求参数
type ListReq struct {
	g.Meta `path:"/" tags:"video" method:"get" summary:"List接口"`
	After  string `json:"after"`
	Limit  int    `json:"limit"`
	Order  string `json:"order"`
}

// List接口响应参数
type ListRes struct {
	g.Meta `mime:"application/json" example:"json"`
	Data   []struct {
		Id     string `json:"id"`
		Object string `json:"object"`
		Model  string `json:"model"`
		Status string `json:"status"`
	} `json:"data"`
	Object string `json:"object"`
}

// Retrieve接口请求参数
type RetrieveReq struct {
	g.Meta  `path:"/{video_id}" tags:"video" method:"get" summary:"Retrieve接口"`
	VideoId string `json:"video_id"`
}

// Retrieve接口响应参数
type RetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Delete接口请求参数
type DeleteReq struct {
	g.Meta  `path:"/{video_id}" tags:"video" method:"delete" summary:"Delete接口"`
	VideoId string `json:"video_id"`
}

// Delete接口响应参数
type DeleteRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Content接口请求参数
type ContentReq struct {
	g.Meta  `path:"/{video_id}/content" tags:"video" method:"get" summary:"Content接口"`
	VideoId string `json:"video_id"`
	Variant string `json:"variant"`
}

// Content接口响应参数
type ContentRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
