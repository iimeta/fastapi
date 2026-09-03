package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// ImageGenerations接口请求参数
type ImageGenerationsReq struct {
	g.Meta `path:"/services/aigc/multimodal-generation/generation" tags:"bailian" method:"post" summary:"ImageGenerations接口"`
	Async  bool `json:"async,omitempty"`
	smodel.BailianImageGenerationReq
}

// ImageGenerations接口响应参数
type ImageGenerationsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoCreate接口请求参数
type VideoCreateReq struct {
	g.Meta `path:"/services/aigc/video-generation/video-synthesis" tags:"bailian" method:"post" summary:"VideoCreate接口"`
	smodel.BailianVideoCreateReq
}

// VideoCreate接口响应参数
type VideoCreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoRetrieve接口请求参数
type VideoRetrieveReq struct {
	g.Meta `path:"/tasks/{task_id}" tags:"bailian" method:"get" summary:"VideoRetrieve接口"`
	smodel.BailianVideoRetrieveReq
}

// VideoRetrieve接口响应参数
type VideoRetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
