// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package file

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/file/v1"
)

type IFileV1 interface {
	Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Retrieve(ctx context.Context, req *v1.RetrieveReq) (res *v1.RetrieveRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error)
}
