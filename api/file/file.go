// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package file

import (
	"context"

	"github.com/iimeta/fastapi/api/file/v1"
)

type IFileV1 interface {
	Files(ctx context.Context, req *v1.FilesReq) (res *v1.FilesRes, err error)
}
