// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package image

import (
	"context"

	"github.com/iimeta/fastapi/api/image/v1"
)

type IImageV1 interface {
	Generations(ctx context.Context, req *v1.GenerationsReq) (res *v1.GenerationsRes, err error)
	Edits(ctx context.Context, req *v1.EditsReq) (res *v1.EditsRes, err error)
}
