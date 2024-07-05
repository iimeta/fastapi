// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package midjourney

import (
	"context"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

type IMidjourneyV1 interface {
	Main(ctx context.Context, req *v1.MainReq) (res *v1.MainRes, err error)
	Fetch(ctx context.Context, req *v1.FetchReq) (res *v1.FetchRes, err error)
}
