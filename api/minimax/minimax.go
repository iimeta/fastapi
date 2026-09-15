// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package minimax

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/minimax/v1"
)

type IMinimaxV1 interface {
	VideoCreate(ctx context.Context, req *v1.VideoCreateReq) (res *v1.VideoCreateRes, err error)
	VideoRetrieve(ctx context.Context, req *v1.VideoRetrieveReq) (res *v1.VideoRetrieveRes, err error)
}
