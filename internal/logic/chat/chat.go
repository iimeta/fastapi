package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/sashabaranov/go-openai"
	"time"
)

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

func (s *sChat) Completions(ctx context.Context, params model.CompletionsReq, retry ...int) (response openai.ChatCompletionResponse, err error) {

	defer func() {
		if err == nil {
			if err = service.Common().RecordUsage(ctx, response.Usage.TotalTokens); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	model, err := service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetKey(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return openai.ChatCompletionResponse{}, err
	}

	key, err := service.Key().PickModelKey(ctx, model.Id)
	if err != nil {
		logger.Error(ctx, err)
		return openai.ChatCompletionResponse{}, err
	}

	client := sdk.NewClient(ctx, model.Model, key.Key, model.BaseUrl)
	response, err = sdk.ChatCompletion(ctx, client, openai.ChatCompletionRequest{
		Model:    model.Model,
		Messages: params.Messages,
	})
	if err != nil {
		logger.Error(ctx, err)
		e := &openai.APIError{}
		if errors.As(err, &e) {

			if len(retry) == 10 {
				response = openai.ChatCompletionResponse{
					ID:      "error",
					Object:  "chat.completion",
					Created: time.Now().Unix(),
					Model:   params.Model,
					Choices: []openai.ChatCompletionChoice{{
						FinishReason: "stop",
						Message: openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleAssistant,
							Content: err.Error(),
						},
					}},
				}
				return
			}

			switch e.HTTPStatusCode {
			case 400:
				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					response = openai.ChatCompletionResponse{
						ID:      "error",
						Object:  "chat.completion",
						Created: time.Now().Unix(),
						Model:   params.Model,
						Choices: []openai.ChatCompletionChoice{{
							FinishReason: "stop",
							Message: openai.ChatCompletionMessage{
								Role:    openai.ChatMessageRoleAssistant,
								Content: err.Error(),
							},
						}},
					}
					return
				}
				response, err = s.Completions(ctx, params, append(retry, 1)...)
			case 429:
				response, err = s.Completions(ctx, params, append(retry, 1)...)
			default:
				response, err = s.Completions(ctx, params, append(retry, 1)...)
			}

			return response, err
		}

		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
}

func (s *sChat) CompletionsStream(ctx context.Context, params model.CompletionsReq, retry ...int) (err error) {

	totalTokens := 0

	defer func() {
		if totalTokens != 0 {
			if err = service.Common().RecordUsage(ctx, totalTokens); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	model, err := service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetKey(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	key, err := service.Key().PickModelKey(ctx, model.Id)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	client := sdk.NewClient(ctx, model.Model, key.Key, model.BaseUrl)
	response, err := sdk.ChatCompletionStream(ctx, client, openai.ChatCompletionRequest{
		Model:    model.Model,
		Messages: params.Messages,
	})

	defer close(response)

	if err != nil {
		logger.Error(ctx, err)
		e := &openai.APIError{}
		if errors.As(err, &e) {

			if len(retry) == 10 {

				response := openai.ChatCompletionStreamResponse{
					ID:      "error",
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   params.Model,
					Choices: []openai.ChatCompletionStreamChoice{{
						FinishReason: "stop",
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content: err.Error(),
						},
					}},
				}

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

			switch e.HTTPStatusCode {
			case 400:
				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {

					response := openai.ChatCompletionStreamResponse{
						ID:      "error",
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   params.Model,
						Choices: []openai.ChatCompletionStreamChoice{{
							FinishReason: "stop",
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Content: err.Error(),
							},
						}},
					}

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
				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			case 429:
				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			default:
				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			}

			return err
		}

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
