// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package bailian

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/bailian/v1"
)

type IBailianV1 interface {
	ImageGenerations(ctx context.Context, req *v1.ImageGenerationsReq) (res *v1.ImageGenerationsRes, err error)
	VideoCreate(ctx context.Context, req *v1.VideoCreateReq) (res *v1.VideoCreateRes, err error)
	VideoRetrieve(ctx context.Context, req *v1.VideoRetrieveReq) (res *v1.VideoRetrieveRes, err error)
}
