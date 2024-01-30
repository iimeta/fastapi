package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
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

func (s *sChat) Completions(ctx context.Context, params model.CompletionsReq, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	var m *model.Model

	defer func() {

		if err == nil {
			if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err = service.Common().RecordUsage(ctx, m, response.Usage); err != nil {
					logger.Error(ctx, err)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			s.SaveChat(ctx, m, params, model.CompletionsRes{
				Completion: response.Choices[0].Message.Content,
				Usage:      response.Usage,
				Error:      err,
				TotalTime:  response.TotalTime,
			})
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return sdkm.ChatCompletionResponse{}, err
	}

	key, err := service.Key().PickModelKey(ctx, m.Id)
	if err != nil {
		logger.Error(ctx, err)
		return sdkm.ChatCompletionResponse{}, err
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, m.BaseUrl)
	response, err = sdk.ChatCompletion(ctx, client, openai.ChatCompletionRequest{
		Model:    m.Model,
		Messages: params.Messages,
	})
	if err != nil {
		logger.Error(ctx, err)
		e := &openai.APIError{}
		if errors.As(err, &e) {

			if len(retry) == 10 {
				response = sdkm.ChatCompletionResponse{
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
					response = sdkm.ChatCompletionResponse{
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

		return sdkm.ChatCompletionResponse{}, err
	}

	return response, nil
}

func (s *sChat) CompletionsStream(ctx context.Context, params model.CompletionsReq, retry ...int) (err error) {

	var m *model.Model
	var usage openai.Usage
	var completion string
	var connTime int64
	var duration int64
	var totalTime int64

	defer func() {

		if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			if usage.TotalTokens != 0 {
				if err = service.Common().RecordUsage(ctx, m, usage); err != nil {
					logger.Error(ctx, err)
				}
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}

		if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			s.SaveChat(ctx, m, params, model.CompletionsRes{
				Completion: completion,
				Usage:      usage,
				Error:      err,
				ConnTime:   connTime,
				Duration:   duration,
				TotalTime:  totalTime,
			})
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	key, err := service.Key().PickModelKey(ctx, m.Id)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, m.BaseUrl)
	response, err := sdk.ChatCompletionStream(ctx, client, openai.ChatCompletionRequest{
		Model:    m.Model,
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

			usage = response.Usage
			completion += response.Choices[0].Delta.Content
			connTime = response.ConnTime
			duration = response.Duration
			totalTime = response.TotalTime

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

func (s *sChat) SaveChat(ctx context.Context, m *model.Model, completionsReq model.CompletionsReq, completionsRes model.CompletionsRes) {

	chat := do.Chat{
		TraceId:         gctx.CtxId(ctx),
		UserId:          service.Session().GetUserId(ctx),
		AppId:           service.Session().GetAppId(ctx),
		Corp:            m.Corp,
		ModelId:         m.Id,
		Name:            m.Name,
		Model:           m.Model,
		Type:            m.Type,
		BaseUrl:         m.BaseUrl,
		Path:            m.Path,
		Proxy:           m.Proxy,
		Stream:          completionsReq.Stream,
		Prompt:          completionsReq.Messages[len(completionsReq.Messages)-1].Content,
		Completion:      completionsRes.Completion,
		PromptRatio:     m.PromptRatio,
		CompletionRatio: m.CompletionRatio,
		ConnTime:        completionsRes.ConnTime,
		Duration:        completionsRes.Duration,
		TotalTime:       completionsRes.TotalTime,
		ReqTime:         g.RequestFromCtx(ctx).EnterTime,
		ReqDate:         gtime.NewFromTimeStamp(g.RequestFromCtx(ctx).EnterTime).Format("Y-m-d"),
		ClientIp:        g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:        g.RequestFromCtx(ctx).GetRemoteIp(),
		Status:          1,
	}

	if completionsRes.Usage.TotalTokens != 0 {
		chat.PromptTokens = int(chat.PromptRatio * float64(completionsRes.Usage.PromptTokens))
		chat.CompletionTokens = int(chat.CompletionRatio * float64(completionsRes.Usage.CompletionTokens))
		chat.TotalTokens = chat.PromptTokens + chat.CompletionTokens
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
		chat.Status = -1
	}

	for _, message := range completionsReq.Messages {
		chat.Messages = append(chat.Messages, do.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
