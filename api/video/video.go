// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package video

import (
	"context"

	"github.com/iimeta/fastapi/api/video/v1"
)

type IVideoV1 interface {
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Remix(ctx context.Context, req *v1.RemixReq) (res *v1.RemixRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Retrieve(ctx context.Context, req *v1.RetrieveReq) (res *v1.RetrieveRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error)
}
