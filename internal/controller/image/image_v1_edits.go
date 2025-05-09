package image

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/iimeta/fastapi/api/image/v1"
)

func (c *ControllerV1) Edits(ctx context.Context, req *v1.EditsReq) (res *v1.EditsRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
