package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
)

type sOpenAI struct{}

func init() {
	service.RegisterOpenAI(New())
}

func New() service.IOpenAI {
	return &sOpenAI{}
}

// Responses
func (s *sOpenAI) Responses(ctx context.Context, request *ghttp.Request, isChatCompletions bool, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.OpenAIResponsesRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sOpenAI Responses time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = common.ConvResponsesToChatCompletionsRequest(request, isChatCompletions)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		chatCompletionResponse := common.ConvResponsesToChatCompletionsResponse(ctx, response)

		if isChatCompletions {
			response.ResponseBytes = gjson.MustEncode(chatCompletionResponse)
		}

		if mak.ReqModel != nil && mak.RealModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				chatCompletionResponse.Model = mak.ReqModel.Model
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ChatCompletionReq: params,
					ChatCompletionRes: chatCompletionResponse,
					Usage:             chatCompletionResponse.Usage,
					Error:             err,
					RetryInfo:         retryInfo,
					ConnTime:          chatCompletionResponse.ConnTime,
					Duration:          chatCompletionResponse.Duration,
					TotalTime:         chatCompletionResponse.TotalTime,
					InternalTime:      internalTime,
					EnterTime:         enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sOpenAI Responses params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]any)
				if err = json.Unmarshal(body, &data); err != nil {
					logger.Error(ctx, err)
					return response, err
				}

				if _, ok := data["model"]; ok {
					data["model"] = mak.RealModel.Model
				}

				body = gjson.MustEncode(data)

				break
			}
		}
	}

	if isChatCompletions {
		body = gjson.MustEncode(common.ConvChatCompletionsToResponsesRequest(ctx, body))
	}

	response, err = common.NewAdapterOpenAI(ctx, mak, false).Responses(ctx, body)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if mak.RealModel.IsEnableModelAgent {
					service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
				} else {
					service.Key().Disabled(ctx, mak.Key, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Responses(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Responses(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, nil, fallbackModel)
						}
					}
				}

				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Responses(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// ResponsesStream
func (s *sOpenAI) ResponsesStream(ctx context.Context, request *ghttp.Request, isChatCompletions bool, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sOpenAI ResponsesStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = common.ConvResponsesToChatCompletionsRequest(request, isChatCompletions)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		completion  string
		serviceTier string
		connTime    int64
		duration    int64
		totalTime   int64
		usage       *smodel.Usage
		retryInfo   *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ChatCompletionReq: params,
					Completion:        completion,
					ServiceTier:       serviceTier,
					Usage:             usage,
					Error:             err,
					RetryInfo:         retryInfo,
					ConnTime:          connTime,
					Duration:          duration,
					TotalTime:         totalTime,
					InternalTime:      internalTime,
					EnterTime:         enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	//request := params
	//
	//if !gstr.Contains(mak.RealModel.Model, "*") {
	//	request.Model = mak.RealModel.Model
	//}
	//
	//// 预设配置
	//if mak.RealModel.IsEnablePresetConfig {
	//
	//	// 替换预设提示词
	//	if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
	//		if request.Messages[0].Role == consts.ROLE_SYSTEM {
	//			request.Messages = append([]smodel.ChatCompletionMessage{{
	//				Role:    consts.ROLE_SYSTEM,
	//				Content: mak.RealModel.PresetConfig.SystemRolePrompt,
	//			}}, request.Messages[1:]...)
	//		} else {
	//			request.Messages = append([]smodel.ChatCompletionMessage{{
	//				Role:    consts.ROLE_SYSTEM,
	//				Content: mak.RealModel.PresetConfig.SystemRolePrompt,
	//			}}, request.Messages...)
	//		}
	//	}
	//
	//	// 检查MaxTokens取值范围
	//	if request.MaxTokens != 0 {
	//		if mak.RealModel.PresetConfig.MinTokens != 0 && request.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
	//			request.MaxTokens = mak.RealModel.PresetConfig.MinTokens
	//		} else if mak.RealModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
	//			request.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
	//		}
	//	}
	//}
	//
	//if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
	//	for i, replaceModel := range mak.ModelAgent.ReplaceModels {
	//		if replaceModel == request.Model {
	//			logger.Infof(ctx, "sOpenAI ResponsesStream request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
	//			request.Model = mak.ModelAgent.TargetModels[i]
	//			mak.RealModel.Model = request.Model
	//			break
	//		}
	//	}
	//}

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sOpenAI ResponsesStream params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]any)
				if err = json.Unmarshal(body, &data); err != nil {
					logger.Error(ctx, err)
					return err
				}

				if _, ok := data["model"]; ok {
					data["model"] = mak.RealModel.Model
				}

				body = gjson.MustEncode(data)

				break
			}
		}
	}

	if isChatCompletions {
		body = gjson.MustEncode(common.ConvChatCompletionsToResponsesRequest(ctx, body))
	}

	response, err := common.NewAdapterOpenAI(ctx, mak, true).ResponsesStream(ctx, body)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if mak.RealModel.IsEnableModelAgent {
					service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
				} else {
					service.Key().Disabled(ctx, mak.Key, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, nil, fallbackModel)
						}
					}
				}

				return err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		res := <-response

		response := common.ConvResponsesStreamToChatCompletionsResponse(ctx, *res)

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Error != nil {

			if errors.Is(response.Error, io.EOF) {

				if response.Usage != nil {
					if usage == nil {
						usage = response.Usage
					} else {
						if response.Usage.PromptTokens != 0 {
							usage.PromptTokens = response.Usage.PromptTokens
						}
						if response.Usage.CompletionTokens != 0 {
							usage.CompletionTokens = response.Usage.CompletionTokens
						}
						if response.Usage.TotalTokens != 0 {
							usage.TotalTokens = response.Usage.TotalTokens
						} else {
							usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						}
						if response.Usage.CacheCreationInputTokens != 0 {
							usage.CacheCreationInputTokens = response.Usage.CacheCreationInputTokens
						}
						if response.Usage.CacheReadInputTokens != 0 {
							usage.CacheReadInputTokens = response.Usage.CacheReadInputTokens
						}
					}
				}

				if err = util.SSEServer(ctx, "[DONE]"); err != nil {
					logger.Error(ctx, err)
					return err
				}

				return nil
			}

			err = response.Error

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			isRetry, isDisabled := common.IsNeedRetry(err)

			if isDisabled {
				if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
					if mak.RealModel.IsEnableModelAgent {
						service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
					} else {
						service.Key().Disabled(ctx, mak.Key, err.Error())
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if isRetry {

				if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

					if mak.RealModel.IsEnableFallback {

						if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
							if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, nil, fallbackModel)
							}
						}
					}

					return err
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				return s.ResponsesStream(g.RequestFromCtx(ctx).GetCtx(), request, isChatCompletions, fallbackModelAgent, fallbackModel, append(retry, 1)...)
			}

			return err
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil {
			if mak.RealModel.Type == 102 && response.Choices[0].Delta.Audio != nil {
				completion += response.Choices[0].Delta.Audio.Transcript
			} else {
				if len(response.Choices) > 1 {
					for i, choice := range response.Choices {
						completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, choice.Delta.Content)
					}
				} else {
					if response.Choices[0].Delta.ReasoningContent != nil {
						completion += gconv.String(response.Choices[0].Delta.ReasoningContent)
					}
					completion += response.Choices[0].Delta.Content
				}
			}
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil && response.Choices[0].Delta.ToolCalls != nil {
			completion += gconv.String(response.Choices[0].Delta.ToolCalls)
		}

		if response.ServiceTier != "" {
			serviceTier = response.ServiceTier
		}

		if response.Usage != nil {
			if usage == nil {
				usage = response.Usage
			} else {
				if response.Usage.PromptTokens != 0 {
					usage.PromptTokens = response.Usage.PromptTokens
				}
				if response.Usage.CompletionTokens != 0 {
					usage.CompletionTokens = response.Usage.CompletionTokens
				}
				if response.Usage.TotalTokens != 0 {
					usage.TotalTokens = response.Usage.TotalTokens
				} else {
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				}
				if response.Usage.CacheCreationInputTokens != 0 {
					usage.CacheCreationInputTokens = response.Usage.CacheCreationInputTokens
				}
				if response.Usage.CacheReadInputTokens != 0 {
					usage.CacheReadInputTokens = response.Usage.CacheReadInputTokens
				}
			}
		}

		data := res.ResponseBytes
		if isChatCompletions {
			data = gjson.MustEncode(response)
		}

		if err = util.SSEServer(ctx, string(data)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}
