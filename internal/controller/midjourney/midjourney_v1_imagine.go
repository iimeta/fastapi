package midjourney

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) Imagine(ctx context.Context, req *v1.ImagineReq) (res *v1.ImagineRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
