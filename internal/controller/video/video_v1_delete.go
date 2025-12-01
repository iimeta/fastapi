package video

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/iimeta/fastapi/api/video/v1"
)

func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	fmt.Println("Delete")
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
