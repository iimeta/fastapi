package chat

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/chat/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if !req.IsToResponses {
		if req.Stream {

			if err = service.Chat().CompletionsStream(ctx, req.ChatCompletionRequest, nil, nil); err != nil {
				return nil, err
			}

			g.RequestFromCtx(ctx).SetCtxVar("stream", req.Stream)

		} else {

			response, err := service.Chat().Completions(ctx, req.ChatCompletionRequest, nil, nil)
			if err != nil {
				return nil, err
			}

			passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
			isResDataPassthrough := passthrough != nil && slices.Contains(passthrough.ResParams, "res_data")

			// 响应头透传
			common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

			if !isResDataPassthrough || response.ResponseBytes == nil {
				g.RequestFromCtx(ctx).Response.WriteJson(response)
			} else {
				g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
			}
		}
	} else {
		if req.Stream {
			if err = service.OpenAI().ResponsesStream(ctx, g.RequestFromCtx(ctx), true, nil, nil); err != nil {
				return nil, err
			}
			g.RequestFromCtx(ctx).SetCtxVar("stream", true)
		} else {
			response, err := service.OpenAI().Responses(ctx, g.RequestFromCtx(ctx), true, nil, nil)
			if err != nil {
				return nil, err
			}

			// 响应头透传
			passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
			common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

			g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
		}
	}

	return
}
