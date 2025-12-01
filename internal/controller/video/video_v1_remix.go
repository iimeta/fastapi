package video

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/iimeta/fastapi/api/video/v1"
)

func (c *ControllerV1) Remix(ctx context.Context, req *v1.RemixReq) (res *v1.RemixRes, err error) {
	fmt.Println("Remix")
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
