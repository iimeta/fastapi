// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package midjourney

import (
	"context"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

type IMidjourneyV1 interface {
	Imagine(ctx context.Context, req *v1.ImagineReq) (res *v1.ImagineRes, err error)
	Change(ctx context.Context, req *v1.ChangeReq) (res *v1.ChangeRes, err error)
	Describe(ctx context.Context, req *v1.DescribeReq) (res *v1.DescribeRes, err error)
	Blend(ctx context.Context, req *v1.BlendReq) (res *v1.BlendRes, err error)
	SwapFace(ctx context.Context, req *v1.SwapFaceReq) (res *v1.SwapFaceRes, err error)
	Action(ctx context.Context, req *v1.ActionReq) (res *v1.ActionRes, err error)
	Modal(ctx context.Context, req *v1.ModalReq) (res *v1.ModalRes, err error)
	Shorten(ctx context.Context, req *v1.ShortenReq) (res *v1.ShortenRes, err error)
	UploadDiscordImages(ctx context.Context, req *v1.UploadDiscordImagesReq) (res *v1.UploadDiscordImagesRes, err error)
	Fetch(ctx context.Context, req *v1.FetchReq) (res *v1.FetchRes, err error)
}
