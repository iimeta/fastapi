// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package volcengine

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/volcengine/v1"
)

type IVolcengineV1 interface {
	VideoCreate(ctx context.Context, req *v1.VideoCreateReq) (res *v1.VideoCreateRes, err error)
	VideoList(ctx context.Context, req *v1.VideoListReq) (res *v1.VideoListRes, err error)
	VideoRetrieve(ctx context.Context, req *v1.VideoRetrieveReq) (res *v1.VideoRetrieveRes, err error)
	VideoDelete(ctx context.Context, req *v1.VideoDeleteReq) (res *v1.VideoDeleteRes, err error)
}
