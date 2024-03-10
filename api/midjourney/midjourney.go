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
	Fetch(ctx context.Context, req *v1.FetchReq) (res *v1.FetchRes, err error)
}
