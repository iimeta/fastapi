package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

type MainReq struct {
	g.Meta `path:"/*" tags:"midjourney" method:"all" summary:"midjourney api"`
}

type MainRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type FetchReq struct {
	g.Meta `path:"/task/:taskId/fetch" tags:"midjourney" method:"get" summary:"midjourney api"`
}

type FetchRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
