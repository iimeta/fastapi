package video

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/iimeta/fastapi/api/video/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	fmt.Println("List")
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
