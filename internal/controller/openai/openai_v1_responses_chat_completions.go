package openai

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/iimeta/fastapi/api/openai/v1"
)

func (c *ControllerV1) ResponsesChatCompletions(ctx context.Context, req *v1.ResponsesChatCompletionsReq) (res *v1.ResponsesChatCompletionsRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
