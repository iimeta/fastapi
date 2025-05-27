package anthropic

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk/anthropic"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/iimeta/go-openai"
	"github.com/iimeta/tiktoken-go"
	"io"
	"math"
)

type sAnthropic struct{}

func init() {
	service.RegisterAnthropic(New())
}

func New() service.IAnthropic {
	return &sAnthropic{}
}

// Completions
func (s *sAnthropic) Completions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

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
		client      *anthropic.Client
		res         sdkm.AnthropicChatCompletionRes
		retryInfo   *mcommon.Retry
		textTokens  int
		imageTokens int
		audioTokens int
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			model := mak.ReqModel.Model

			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if mak.ReqModel.Type == 100 { // 多模态

				if response.Usage == nil {

					response.Usage = new(sdkm.Usage)

					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, mak.ReqModel)
						response.Usage.PromptTokens = textTokens + imageTokens
					} else {
						if response.Usage.PromptTokens == 0 {
							response.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
						}
					}

					if response.Usage.CompletionTokens == 0 && len(response.Choices) > 0 && response.Choices[0].Message != nil {
						for _, choice := range response.Choices {
							response.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
						}
					}

					response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

				} else {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
				}

				if response.Usage.CacheCreationInputTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
				}

				if response.Usage.CacheReadInputTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
				}

			} else if mak.ReqModel.Type == 102 { // 多模态语音

				if response.Usage == nil {

					response.Usage = new(sdkm.Usage)

					textTokens, audioTokens = common.GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
					response.Usage.PromptTokens = textTokens + audioTokens

					if len(response.Choices) > 0 && response.Choices[0].Message != nil && response.Choices[0].Message.Audio != nil {
						for _, choice := range response.Choices {
							response.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, choice.Message.Audio.Transcript) + 388
						}
					}
				}

				response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
				totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

			} else if response.Usage == nil || response.Usage.TotalTokens == 0 {

				response.Usage = new(sdkm.Usage)

				response.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)

				if len(response.Choices) > 0 && response.Choices[0].Message != nil {
					for _, choice := range response.Choices {
						response.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
					}
				}

				response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
			}
		}

		if mak.ReqModel != nil && response.Usage != nil {
			if mak.ReqModel.Type == 102 {

				if response.Usage.PromptTokensDetails != nil {
					textTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
					audioTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				} else {
					audioTokens = int(math.Ceil(float64(response.Usage.PromptTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				}

				if response.Usage.CompletionTokensDetails != nil {
					textTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
					audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				} else {
					audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				}

				totalTokens = textTokens + audioTokens

			} else if mak.ReqModel.Type != 100 {
				if mak.ReqModel.TextQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))
				} else {
					totalTokens = mak.ReqModel.TextQuota.FixedQuota
				}
			}
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key, mak.Group); err != nil {
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
					completionsRes.Usage.TotalTokens = totalTokens
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

				service.Chat().SaveLog(ctx, mak.Group, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo, false)

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
	//			request.Messages = append([]sdkm.ChatCompletionMessage{{
	//				Role:    consts.ROLE_SYSTEM,
	//				Content: mak.RealModel.PresetConfig.SystemRolePrompt,
	//			}}, request.Messages[1:]...)
	//		} else {
	//			request.Messages = append([]sdkm.ChatCompletionMessage{{
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

	if client, err = common.NewAnthropicClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	res, err = client.ChatCompletionOfficial(ctx, request.GetBody())
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if mak.RealModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, mak.Key, err.Error())
				} else {
					service.Key().DisabledModelKey(ctx, mak.Key, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); fallbackModelAgent != nil {
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

	response = convToChatCompletionResponse(g.RequestFromCtx(ctx).GetCtx(), res, false)

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
		client      *anthropic.Client
		completion  string
		connTime    int64
		duration    int64
		totalTime   int64
		textTokens  int
		imageTokens int
		audioTokens int
		totalTokens int
		usage       *sdkm.Usage
		retryInfo   *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			if retryInfo == nil && completion != "" && (usage == nil || usage.PromptTokens == 0 || usage.CompletionTokens == 0) && mak.ReqModel != nil {

				if usage == nil {
					usage = new(sdkm.Usage)
				}

				model := mak.ReqModel.Model
				if !tiktoken.IsEncodingForModel(model) {
					model = consts.DEFAULT_MODEL
				}

				if mak.ReqModel.Type == 102 { // 多模态语音
					textTokens, audioTokens = common.GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
					usage.PromptTokens = textTokens + audioTokens
				} else {
					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, mak.ReqModel)
						usage.PromptTokens = textTokens + imageTokens
					} else {
						if usage.PromptTokens == 0 {
							usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
						}
					}
				}

				if usage.CompletionTokens == 0 {
					usage.CompletionTokens = common.GetCompletionTokens(ctx, model, completion)
					if mak.ReqModel.Type == 102 { // 多模态语音
						usage.CompletionTokens += 388
					}
				}

				if mak.ReqModel.Type == 100 { // 多模态

					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

					if usage.CacheCreationInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
					}

					if usage.CacheReadInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
					}

				} else if mak.ReqModel.Type == 102 { // 多模态语音
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				} else {
					if mak.ReqModel.TextQuota.BillingMethod == 1 {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))
					} else {
						usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
						totalTokens = mak.ReqModel.TextQuota.FixedQuota
					}
				}

			} else if retryInfo == nil && usage != nil && mak.ReqModel != nil {

				if mak.ReqModel.Type == 100 { // 多模态

					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

					if usage.CacheCreationInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
					}

					if usage.CacheReadInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
					}

				} else if mak.ReqModel.Type == 102 { // 多模态语音
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				} else {
					if mak.ReqModel.TextQuota.BillingMethod == 1 {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))
					} else {
						usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
						totalTokens = mak.ReqModel.TextQuota.FixedQuota
					}
				}
			}

			if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {
				if err := grpool.Add(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key, mak.Group); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}
			}

			if mak.ReqModel != nil && mak.RealModel != nil {
				if err := grpool.Add(ctx, func(ctx context.Context) {

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
						completionsRes.Usage.TotalTokens = totalTokens
					}

					service.Chat().SaveLog(ctx, mak.Group, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo, false)

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
	//			request.Messages = append([]sdkm.ChatCompletionMessage{{
	//				Role:    consts.ROLE_SYSTEM,
	//				Content: mak.RealModel.PresetConfig.SystemRolePrompt,
	//			}}, request.Messages[1:]...)
	//		} else {
	//			request.Messages = append([]sdkm.ChatCompletionMessage{{
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

	if client, err = common.NewAnthropicClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return err
	}

	response, err := client.ChatCompletionStreamOfficial(ctx, request.GetBody())
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if mak.RealModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, mak.Key, err.Error())
				} else {
					service.Key().DisabledModelKey(ctx, mak.Key, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); fallbackModelAgent != nil {
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

		res := <-response

		response := convToChatCompletionResponse(g.RequestFromCtx(ctx).GetCtx(), *res, true)

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
						service.ModelAgent().DisabledModelAgentKey(ctx, mak.Key, err.Error())
					} else {
						service.Key().DisabledModelKey(ctx, mak.Key, err.Error())
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if isRetry {

				if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

					if mak.RealModel.IsEnableFallback {

						if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
							if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); fallbackModelAgent != nil {
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
			}
		}

		data := make(map[string]interface{})
		if err = gjson.Unmarshal(response.ResponseBytes, &data); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = util.SSEServer(ctx, gjson.MustEncodeString(data)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

func convToChatCompletionRequest(request *ghttp.Request) sdkm.ChatCompletionRequest {

	anthropicChatCompletionReq := sdkm.AnthropicChatCompletionReq{}
	if err := gjson.Unmarshal(request.GetBody(), &anthropicChatCompletionReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return sdkm.ChatCompletionRequest{}
	}

	messages := make([]sdkm.ChatCompletionMessage, 0)
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

			messages = append(messages, sdkm.ChatCompletionMessage{
				Role:    consts.ROLE_USER,
				Content: contents,
			})

		} else {
			messages = append(messages, message)
		}
	}

	if anthropicChatCompletionReq.System != nil {
		messages = append([]sdkm.ChatCompletionMessage{{
			Role:    consts.ROLE_SYSTEM,
			Content: anthropicChatCompletionReq.System,
		}}, messages...)
	}

	return sdkm.ChatCompletionRequest{
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

func convToChatCompletionResponse(ctx context.Context, res sdkm.AnthropicChatCompletionRes, stream bool) sdkm.ChatCompletionResponse {

	anthropicChatCompletionRes := sdkm.AnthropicChatCompletionRes{
		ResponseBytes: res.ResponseBytes,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &anthropicChatCompletionRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := sdkm.ChatCompletionResponse{
		ID:            consts.COMPLETION_ID_PREFIX + anthropicChatCompletionRes.Id,
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
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
				Delta: &sdkm.ChatCompletionStreamChoiceDelta{
					Role: consts.ROLE_ASSISTANT,
					ToolCalls: []openai.ToolCall{{
						Function: openai.FunctionCall{
							Arguments: anthropicChatCompletionRes.Delta.PartialJson,
						},
					}},
				},
			})
		} else {
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
				Delta: &sdkm.ChatCompletionStreamChoiceDelta{
					Role:    consts.ROLE_ASSISTANT,
					Content: anthropicChatCompletionRes.Delta.Text,
				},
			})
		}
	} else if len(anthropicChatCompletionRes.Content) > 0 {
		for _, content := range anthropicChatCompletionRes.Content {
			if content.Type == consts.DELTA_TYPE_INPUT_JSON {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
					Delta: &sdkm.ChatCompletionStreamChoiceDelta{
						Role: consts.ROLE_ASSISTANT,
						ToolCalls: []openai.ToolCall{{
							Function: openai.FunctionCall{
								Arguments: content.PartialJson,
							},
						}},
					},
				})
			} else {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
					Message: &sdkm.ChatCompletionMessage{
						Role:    anthropicChatCompletionRes.Role,
						Content: content.Text,
					},
					FinishReason: "stop",
				})
			}
		}
	}

	if anthropicChatCompletionRes.Message.Usage != nil {
		chatCompletionResponse.Usage = &sdkm.Usage{
			PromptTokens:             anthropicChatCompletionRes.Message.Usage.InputTokens,
			CompletionTokens:         anthropicChatCompletionRes.Message.Usage.OutputTokens,
			TotalTokens:              anthropicChatCompletionRes.Message.Usage.InputTokens + anthropicChatCompletionRes.Message.Usage.OutputTokens,
			CacheCreationInputTokens: anthropicChatCompletionRes.Message.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     anthropicChatCompletionRes.Message.Usage.CacheReadInputTokens,
		}
	}

	if anthropicChatCompletionRes.Usage != nil {
		chatCompletionResponse.Usage = &sdkm.Usage{
			PromptTokens:             anthropicChatCompletionRes.Usage.InputTokens,
			CompletionTokens:         anthropicChatCompletionRes.Usage.OutputTokens,
			TotalTokens:              anthropicChatCompletionRes.Usage.InputTokens + anthropicChatCompletionRes.Usage.OutputTokens,
			CacheCreationInputTokens: anthropicChatCompletionRes.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     anthropicChatCompletionRes.Usage.CacheReadInputTokens,
		}
	}

	return chatCompletionResponse
}
