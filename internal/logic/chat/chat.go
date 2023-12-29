package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/iimeta/iim-sdk/sdk"
	"github.com/sashabaranov/go-openai"
	"reflect"
)

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

func (s *sChat) Completions(ctx context.Context, params model.CompletionsReq) (response openai.ChatCompletionResponse, err error) {

	defer func() {
		if err == nil {
			if err = service.Common().RecordUsage(ctx, response.Usage.TotalTokens); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	chat := sdk.NewChat()
	chat.Corp = sdk.CORP_OPENAI
	chat.Model = params.Model
	chat.Messages = params.Messages

	response, err = sdk.Chat.Chat(ctx, chat)
	if err != nil {
		e := &openai.APIError{}
		if errors.As(err, &e) && !reflect.DeepEqual(response, openai.ChatCompletionResponse{}) {
			return response, nil
		}
		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
}

func (s *sChat) CompletionsStream(ctx context.Context, params model.CompletionsReq) (err error) {

	totalTokens := 0

	defer func() {
		if totalTokens != 0 {
			if err = service.Common().RecordUsage(ctx, totalTokens); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	chat := sdk.NewChat()
	chat.Corp = sdk.CORP_OPENAI
	chat.Model = params.Model
	chat.Messages = params.Messages

	response, err := sdk.Chat.ChatStream(ctx, chat)
	defer close(response)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	for {
		select {
		case response := <-response:

			totalTokens = response.Usage.TotalTokens

			if response.Choices[0].FinishReason == "stop" {

				if err = util.SSEServer(ctx, "", gjson.MustEncode(response)); err != nil {
					logger.Error(ctx, err)
					return err
				}

				if err = util.SSEServer(ctx, "", "[DONE]"); err != nil {
					logger.Error(ctx, err)
					return err
				}

				return nil
			}

			if err = util.SSEServer(ctx, "", gjson.MustEncode(response)); err != nil {
				logger.Error(ctx, err)
				return err
			}
		default:
			if err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}
}
