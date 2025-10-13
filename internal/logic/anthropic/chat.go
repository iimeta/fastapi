package anthropic

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
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sAnthropic struct{}

func init() {
	service.RegisterAnthropic(New())
}

func New() service.IAnthropic {
	return &sAnthropic{}
}

// Completions
func (s *sAnthropic) Completions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAnthropic Completions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convToChatCompletionRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			completion := ""
			if len(response.Choices) > 0 && response.Choices[0].Message != nil {
				for _, choice := range response.Choices {
					completion += gconv.String(choice.Message.Content)
				}
			}

			billingData := &mcommon.BillingData{
				ChatCompletionRequest: params,
				Completion:            completion,
				Usage:                 response.Usage,
			}

			// 计算花费
			spend = common.Billing(ctx, mak, billingData)
			response.Usage = billingData.Usage

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := common.RecordSpend(ctx, spend, mak); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				completionsRes := &model.CompletionsRes{
					Error:        err,
					ConnTime:     response.ConnTime,
					Duration:     response.Duration,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && response.Usage != nil {
					completionsRes.Usage = *response.Usage
					completionsRes.Usage.TotalTokens = spend.TotalSpendTokens
				}

				if retryInfo == nil && len(response.Choices) > 0 && response.Choices[0].Message != nil {
					if mak.RealModel.Type == 102 && response.Choices[0].Message.Audio != nil {
						completionsRes.Completion = response.Choices[0].Message.Audio.Transcript
					} else {
						if len(response.Choices) > 1 {
							for i, choice := range response.Choices {
								completionsRes.Completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, gconv.String(choice.Message.Content))
							}
						} else {
							completionsRes.Completion = gconv.String(response.Choices[0].Message.Content)
						}
					}
				}

				service.Chat().SaveLog(ctx, model.ChatLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					CompletionsReq:     &params,
					CompletionsRes:     completionsRes,
					RetryInfo:          retryInfo,
					Spend:              spend,
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

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sAnthropic Completions params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]interface{})
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

	response, err = common.NewAdapter(ctx, mak, false, true).ChatCompletions(ctx, body)
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
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
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

			return s.Completions(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// CompletionsStream
func (s *sAnthropic) CompletionsStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAnthropic CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convToChatCompletionRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		completion string
		connTime   int64
		duration   int64
		totalTime  int64
		usage      *smodel.Usage
		retryInfo  *mcommon.Retry
		spend      mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

				billingData := &mcommon.BillingData{
					ChatCompletionRequest: params,
					Completion:            completion,
					Usage:                 usage,
				}

				// 计算花费
				spend = common.Billing(ctx, mak, billingData)
				usage = billingData.Usage

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
					if err := common.RecordSpend(ctx, spend, mak); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}
			}

			if mak.ReqModel != nil && mak.RealModel != nil {
				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					completionsRes := &model.CompletionsRes{
						Completion:   completion,
						Error:        err,
						ConnTime:     connTime,
						Duration:     duration,
						TotalTime:    totalTime,
						InternalTime: internalTime,
						EnterTime:    enterTime,
					}

					if usage != nil {
						completionsRes.Usage = *usage
						completionsRes.Usage.TotalTokens = spend.TotalSpendTokens
					}

					service.Chat().SaveLog(ctx, model.ChatLog{
						ReqModel:           mak.ReqModel,
						RealModel:          mak.RealModel,
						ModelAgent:         mak.ModelAgent,
						FallbackModelAgent: fallbackModelAgent,
						FallbackModel:      fallbackModel,
						Key:                mak.Key,
						CompletionsReq:     &params,
						CompletionsRes:     completionsRes,
						RetryInfo:          retryInfo,
						Spend:              spend,
					})

				}); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}

		}); err != nil {
			logger.Error(ctx, err)
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

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sAnthropic CompletionsStream params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]interface{})
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

	response, err := common.NewAdapter(ctx, mak, true, true).ChatCompletionsStream(ctx, body)
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
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
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

			return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Error != nil {

			if errors.Is(response.Error, io.EOF) {

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
								return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
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

				return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
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
					completion += response.Choices[0].Delta.Content
				}
			}
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil && response.Choices[0].Delta.ToolCalls != nil {
			completion += gconv.String(response.Choices[0].Delta.ToolCalls)
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

		if err = util.SSEServer(ctx, string(response.ResponseBytes)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

func convToChatCompletionRequest(request *ghttp.Request) smodel.ChatCompletionRequest {

	anthropicChatCompletionReq := smodel.AnthropicChatCompletionReq{}
	if err := gjson.Unmarshal(request.GetBody(), &anthropicChatCompletionReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return smodel.ChatCompletionRequest{}
	}

	messages := make([]smodel.ChatCompletionMessage, 0)
	for _, message := range anthropicChatCompletionReq.Messages {

		if contents, ok := message.Content.([]interface{}); ok {

			for _, value := range contents {
				if content, ok := value.(map[string]interface{}); ok {
					if content["type"] == "image" {
						if source, ok := content["source"].(map[string]interface{}); ok {
							if source["data"] != nil {
								content["type"] = "image_url"
								content["image_url"] = g.MapStrAny{
									"url": source["data"],
								}
								delete(content, "source")
							}
						}
					}
				}
			}

			messages = append(messages, smodel.ChatCompletionMessage{
				Role:    sconsts.ROLE_USER,
				Content: contents,
			})

		} else {
			messages = append(messages, message)
		}
	}

	if anthropicChatCompletionReq.System != nil {
		messages = append([]smodel.ChatCompletionMessage{{
			Role:    sconsts.ROLE_SYSTEM,
			Content: anthropicChatCompletionReq.System,
		}}, messages...)
	}

	return smodel.ChatCompletionRequest{
		Model:       anthropicChatCompletionReq.Model,
		Messages:    messages,
		MaxTokens:   anthropicChatCompletionReq.MaxTokens,
		Stream:      anthropicChatCompletionReq.Stream,
		Temperature: anthropicChatCompletionReq.Temperature,
		ToolChoice:  anthropicChatCompletionReq.ToolChoice,
		Tools:       anthropicChatCompletionReq.Tools,
		TopK:        anthropicChatCompletionReq.TopK,
		TopP:        anthropicChatCompletionReq.TopP,
	}
}

func convToChatCompletionResponse(ctx context.Context, res smodel.AnthropicChatCompletionRes, stream bool) smodel.ChatCompletionResponse {

	anthropicChatCompletionRes := smodel.AnthropicChatCompletionRes{
		ResponseBytes: res.ResponseBytes,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &anthropicChatCompletionRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := smodel.ChatCompletionResponse{
		Id:            consts.COMPLETION_ID_PREFIX + anthropicChatCompletionRes.Id,
		Object:        consts.COMPLETION_OBJECT,
		Created:       gtime.Timestamp(),
		Model:         anthropicChatCompletionRes.Model,
		ResponseBytes: res.ResponseBytes,
		ConnTime:      res.ConnTime,
		Duration:      res.Duration,
		TotalTime:     res.TotalTime,
		Error:         anthropicChatCompletionRes.Err,
	}

	if stream {
		if anthropicChatCompletionRes.Delta.Type == consts.DELTA_TYPE_INPUT_JSON {
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
				Delta: &smodel.ChatCompletionStreamChoiceDelta{
					Role: sconsts.ROLE_ASSISTANT,
					ToolCalls: []smodel.ToolCall{{
						Function: smodel.FunctionCall{
							Arguments: anthropicChatCompletionRes.Delta.PartialJson,
						},
					}},
				},
			})
		} else {
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
				Delta: &smodel.ChatCompletionStreamChoiceDelta{
					Role:    sconsts.ROLE_ASSISTANT,
					Content: anthropicChatCompletionRes.Delta.Text,
				},
			})
		}
	} else if len(anthropicChatCompletionRes.Content) > 0 {
		for _, content := range anthropicChatCompletionRes.Content {
			if content.Type == consts.DELTA_TYPE_INPUT_JSON {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
					Delta: &smodel.ChatCompletionStreamChoiceDelta{
						Role: sconsts.ROLE_ASSISTANT,
						ToolCalls: []smodel.ToolCall{{
							Function: smodel.FunctionCall{
								Arguments: content.PartialJson,
							},
						}},
					},
				})
			} else {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
					Message: &smodel.ChatCompletionMessage{
						Role:    anthropicChatCompletionRes.Role,
						Content: content.Text,
					},
					FinishReason: "stop",
				})
			}
		}
	}

	if anthropicChatCompletionRes.Message.Usage != nil {
		chatCompletionResponse.Usage = &smodel.Usage{
			PromptTokens:             anthropicChatCompletionRes.Message.Usage.InputTokens,
			CompletionTokens:         anthropicChatCompletionRes.Message.Usage.OutputTokens,
			TotalTokens:              anthropicChatCompletionRes.Message.Usage.InputTokens + anthropicChatCompletionRes.Message.Usage.OutputTokens,
			CacheCreationInputTokens: anthropicChatCompletionRes.Message.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     anthropicChatCompletionRes.Message.Usage.CacheReadInputTokens,
		}
	}

	if anthropicChatCompletionRes.Usage != nil {
		chatCompletionResponse.Usage = &smodel.Usage{
			PromptTokens:             anthropicChatCompletionRes.Usage.InputTokens,
			CompletionTokens:         anthropicChatCompletionRes.Usage.OutputTokens,
			TotalTokens:              anthropicChatCompletionRes.Usage.InputTokens + anthropicChatCompletionRes.Usage.OutputTokens,
			CacheCreationInputTokens: anthropicChatCompletionRes.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     anthropicChatCompletionRes.Usage.CacheReadInputTokens,
		}
	}

	return chatCompletionResponse
}
