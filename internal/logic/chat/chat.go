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
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/sashabaranov/go-openai"
)

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

func (s *sChat) Completions(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Completions time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl string

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, m, response.Usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				completionsRes := model.CompletionsRes{
					Usage:        response.Usage,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if len(response.Choices) > 0 {
					completionsRes.Completion = response.Choices[0].Message.Content
				}

				s.SaveChat(ctx, m, key, params, completionsRes)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if m.IsEnableModelAgent {

		if modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				logger.Error(ctx, err)
				return response, err
			}
		}

	} else {
		if key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, baseUrl)
	if response, err = sdk.ChatCompletion(ctx, client, params); err != nil {
		logger.Error(ctx, err)

		if len(retry) == 10 {
			return response, err
		}

		e := &openai.APIError{}
		if errors.As(err, &e) {

			switch e.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return response, err
				}

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			case 429:

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			default:

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)
			}

			return response, err
		}

		return response, err
	}

	return response, nil
}

func (s *sChat) CompletionsStream(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl string
	var completion string
	var connTime int64
	var duration int64
	var totalTime int64

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			usage := openai.Usage{}

			if completion != "" {

				numTokensFromMessagesTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(m.Model, params.Messages); err != nil {
					logger.Errorf(ctx, "CompletionsStream model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					usage.PromptTokens = promptTokens
				}
				logger.Debugf(ctx, "NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-numTokensFromMessagesTime)

				completionTime := gtime.TimestampMilli()
				if completionTokens, err := tiktoken.NumTokensFromString(m.Model, completion); err != nil {
					logger.Errorf(ctx, "CompletionsStream model: %s, completion: %s, NumTokensFromString error: %v", params.Model, completion, err)
				} else {
					logger.Debugf(ctx, "NumTokensFromString len(completion): %d, time: %d", len(completion), gtime.TimestampMilli()-completionTime)

					usage.CompletionTokens = completionTokens
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

					if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
						if err := service.Common().RecordUsage(ctx, m, usage); err != nil {
							logger.Error(ctx, err)
						}
					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
				m.ModelAgent = modelAgent
				s.SaveChat(ctx, m, key, params, model.CompletionsRes{
					Completion:   completion,
					Usage:        usage,
					Error:        err,
					ConnTime:     connTime,
					Duration:     duration,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				})
			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if m.IsEnableModelAgent {

		if modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}

	} else {
		if key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, baseUrl)
	response, err := sdk.ChatCompletionStream(ctx, client, params)
	if err != nil {
		logger.Error(ctx, err)

		if len(retry) == 10 {
			return err
		}

		e := &openai.APIError{}
		if errors.As(err, &e) {

			switch e.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return err
				}

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			case 429:

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			default:

				if m.IsEnableModelAgent {
					service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
				} else {
					service.Key().RecordErrorModelKey(ctx, m, key)
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			}

			return err
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		if response == nil {
			return nil
		}

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
	}
}

func (s *sChat) SaveChat(ctx context.Context, model *model.Model, key *model.Key, completionsReq openai.ChatCompletionRequest, completionsRes model.CompletionsRes) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Stream:       completionsReq.Stream,
		Prompt:       completionsReq.Messages[len(completionsReq.Messages)-1].Content,
		Completion:   completionsRes.Completion,
		ConnTime:     completionsRes.ConnTime,
		Duration:     completionsRes.Duration,
		TotalTime:    completionsRes.TotalTime,
		InternalTime: completionsRes.InternalTime,
		ReqTime:      completionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		Status:       1,
	}

	if model != nil {
		chat.Corp = model.Corp
		chat.ModelId = model.Id
		chat.Name = model.Name
		chat.Model = model.Model
		chat.Type = model.Type
		chat.PromptRatio = model.PromptRatio
		chat.CompletionRatio = model.CompletionRatio
		chat.IsEnableModelAgent = model.IsEnableModelAgent
		if chat.IsEnableModelAgent {
			chat.ModelAgentId = model.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
				Name:    model.ModelAgent.Name,
				BaseUrl: model.ModelAgent.BaseUrl,
				Path:    model.ModelAgent.Path,
				Weight:  model.ModelAgent.Weight,
				Remark:  model.ModelAgent.Remark,
				Status:  model.ModelAgent.Status,
			}
		}
	}

	if key != nil {
		chat.Key = key.Key
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
