// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package midjourney

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/midjourney/v1"
)

type IMidjourneyV1 interface {
	Submit(ctx context.Context, req *v1.SubmitReq) (res *v1.SubmitRes, err error)
	ModelSubmit(ctx context.Context, req *v1.ModelSubmitReq) (res *v1.ModelSubmitRes, err error)
	Task(ctx context.Context, req *v1.TaskReq) (res *v1.TaskRes, err error)
	ModelTask(ctx context.Context, req *v1.ModelTaskReq) (res *v1.ModelTaskRes, err error)
}
