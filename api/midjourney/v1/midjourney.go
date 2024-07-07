package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

type SubmitReq struct {
	g.Meta `path:"/submit/*" tags:"midjourney" method:"all" summary:"midjourney api"`
}

type SubmitRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ModelSubmitReq struct {
	g.Meta `path:"/:model/submit/*" tags:"midjourney" method:"all" summary:"midjourney api"`
}

type ModelSubmitRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type TaskReq struct {
	g.Meta `path:"/task/*" tags:"midjourney" method:"all" summary:"midjourney api"`
}

type TaskRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ModelTaskReq struct {
	g.Meta `path:"/:model/task/*" tags:"midjourney" method:"all" summary:"midjourney api"`
}

type ModelTaskRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
