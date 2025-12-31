package health

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/health/v1"
)

func (c *ControllerV1) Health(ctx context.Context, req *v1.HealthReq) (res *v1.HealthRes, err error) {
	return
}
