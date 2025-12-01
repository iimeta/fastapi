package video

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/iimeta/fastapi/api/video/v1"
)

func (c *ControllerV1) Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error) {
	fmt.Println("Content")
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
