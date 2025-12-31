// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package batch

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/batch/v1"
)

type IBatchV1 interface {
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Retrieve(ctx context.Context, req *v1.RetrieveReq) (res *v1.RetrieveRes, err error)
	Cancel(ctx context.Context, req *v1.CancelReq) (res *v1.CancelRes, err error)
}
