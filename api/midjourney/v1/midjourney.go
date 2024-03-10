package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/model"
)

type ImagineReq struct {
	g.Meta `path:"/submit/imagine" tags:"midjourney" method:"post" summary:"midjourney api"`
	model.MidjourneyProxyImagineReq
}

type ImagineRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ChangeReq struct {
	g.Meta `path:"/submit/change" tags:"midjourney" method:"post" summary:"midjourney api"`
	model.MidjourneyProxyChangeReq
}

type ChangeRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type DescribeReq struct {
	g.Meta `path:"/submit/describe" tags:"midjourney" method:"post" summary:"midjourney api"`
	model.MidjourneyProxyDescribeReq
}

type DescribeRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type BlendReq struct {
	g.Meta `path:"/submit/blend" tags:"midjourney" method:"post" summary:"midjourney api"`
	model.MidjourneyProxyBlendReq
}

type BlendRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type FetchReq struct {
	g.Meta `path:"/task/:task_id/fetch" tags:"midjourney" method:"get" summary:"midjourney api"`
	TaskId string `json:"task_id"`
}

type FetchRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
