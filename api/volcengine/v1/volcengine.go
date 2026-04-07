package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// VideoCreate接口请求参数
type VideoCreateReq struct {
	g.Meta `path:"/tasks" tags:"volcengine" method:"post" summary:"VideoCreate接口"`
	smodel.VolcVideoCreateReq
}

// VideoCreate接口响应参数
type VideoCreateRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoList接口请求参数
type VideoListReq struct {
	g.Meta `path:"/tasks" tags:"volcengine" method:"get" summary:"VideoList接口"`
	smodel.VolcVideoListReq
}

// VideoList接口响应参数
type VideoListRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoRetrieve接口请求参数
type VideoRetrieveReq struct {
	g.Meta `path:"/tasks/{task_id}" tags:"volcengine" method:"get" summary:"VideoRetrieve接口"`
	smodel.VolcVideoRetrieveReq
}

// VideoRetrieve接口响应参数
type VideoRetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// VideoDelete接口请求参数
type VideoDeleteReq struct {
	g.Meta `path:"/tasks/{task_id}" tags:"volcengine" method:"delete" summary:"VideoDelete接口"`
	smodel.VolcVideoDeleteReq
}

// VideoDelete接口响应参数
type VideoDeleteRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
