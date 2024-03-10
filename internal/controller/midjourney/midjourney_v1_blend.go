package midjourney

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) Blend(ctx context.Context, req *v1.BlendReq) (res *v1.BlendRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
