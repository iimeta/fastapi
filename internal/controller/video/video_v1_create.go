package video

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/iimeta/fastapi/api/video/v1"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	fmt.Println("Create")
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
