// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package image

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/image/v1"
)

type IImageV1 interface {
	Generations(ctx context.Context, req *v1.GenerationsReq) (res *v1.GenerationsRes, err error)
	Edits(ctx context.Context, req *v1.EditsReq) (res *v1.EditsRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Retrieve(ctx context.Context, req *v1.RetrieveReq) (res *v1.RetrieveRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error)
}
