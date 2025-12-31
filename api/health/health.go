// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package health

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/health/v1"
)

type IHealthV1 interface {
	Health(ctx context.Context, req *v1.HealthReq) (res *v1.HealthRes, err error)
}
