package chat

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/iimeta/fastapi/api/chat/v1"
)

func (c *ControllerV1) CompletionsResponses(ctx context.Context, req *v1.CompletionsResponsesReq) (res *v1.CompletionsResponsesRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
