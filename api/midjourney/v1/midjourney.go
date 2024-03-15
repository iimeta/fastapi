package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

type ImagineReq struct {
	g.Meta `path:"/submit/imagine" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type ImagineRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ChangeReq struct {
	g.Meta `path:"/submit/change" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type ChangeRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type DescribeReq struct {
	g.Meta `path:"/submit/describe" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type DescribeRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type BlendReq struct {
	g.Meta `path:"/submit/blend" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type BlendRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type SwapFaceReq struct {
	g.Meta `path:"/insight-face/swap" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type SwapFaceRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ActionReq struct {
	g.Meta `path:"/submit/action" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type ActionRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ModalReq struct {
	g.Meta `path:"/submit/modal" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type ModalRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type ShortenReq struct {
	g.Meta `path:"/submit/shorten" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type ShortenRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type UploadDiscordImagesReq struct {
	g.Meta `path:"/submit/upload-discord-images" tags:"midjourney" method:"post" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type UploadDiscordImagesRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

type FetchReq struct {
	g.Meta `path:"/task/:taskId/fetch" tags:"midjourney" method:"get" summary:"midjourney api"`
	sdkm.MidjourneyProxyRequest
}

type FetchRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
