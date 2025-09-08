package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/model"
)

// Moderations接口请求参数
type ModerationsReq struct {
	g.Meta `path:"/moderations" tags:"moderation" method:"post" summary:"moderations接口"`
	smodel.ModerationRequest
}

// Moderations接口响应参数
type ModerationsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
