package chat

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/iimeta/tiktoken-go"
	"io"
	"math"
	"slices"
	"sync"
	"time"
)

type sChat struct {
	mutex sync.Mutex
}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

// Completions
func (s *sChat) Completions(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if len(params.Functions) == 0 {
		params.Messages = common.HandleMessages(params.Messages)
		if len(params.Messages) == 0 {
			return response, errors.ERR_INVALID_PARAMETER
		}
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      sdk.Client
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
			response.Model = mak.ReqModel.Model
			model := mak.ReqModel.Model

			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if mak.ReqModel.Type == 100 { // 多模态

				if response.Usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

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

				if params.Tools != nil {
					if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
						totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
						response.Usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
					}
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
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key); err != nil {
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
							if response.Choices[0].Message.ReasoningContent != nil {
								completionsRes.Completion = gconv.String(response.Choices[0].Message.ReasoningContent)
							}
							completionsRes.Completion += gconv.String(response.Choices[0].Message.Content)
						}
					}
				}

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo, false)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	request := params

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if request.Messages[0].Role == consts.ROLE_SYSTEM {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages[1:]...)
			} else {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if request.MaxTokens != 0 {
			if mak.RealModel.PresetConfig.MinTokens != 0 && request.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MinTokens
			} else if mak.RealModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
			}
		}
	}

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.ChatCompletion(ctx, request)
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
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// CompletionsStream
func (s *sChat) CompletionsStream(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	if len(params.Functions) == 0 {
		params.Messages = common.HandleMessages(params.Messages)
		if len(params.Messages) == 0 {
			return errors.ERR_INVALID_PARAMETER
		}
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      sdk.Client
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
			if retryInfo == nil && completion != "" && mak.ReqModel != nil && (usage == nil || usage.PromptTokens == 0 || usage.CompletionTokens == 0 || (mak.ReqModel.Type == 100 && mak.ReqModel.MultimodalQuota.BillingRule == 2)) {

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

					if params.Tools != nil {
						if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
							totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
							usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
						}
					}

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

					if params.Tools != nil {
						if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
							totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
							usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
						}
					}

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
					if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key); err != nil {
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

					s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo, false)

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

	request := params

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if request.Messages[0].Role == consts.ROLE_SYSTEM {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages[1:]...)
			} else {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if request.MaxTokens != 0 {
			if mak.RealModel.PresetConfig.MinTokens != 0 && request.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MinTokens
			} else if mak.RealModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
			}
		}
	}

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return err
	}

	response, err := client.ChatCompletionStream(ctx, request)
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
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
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
								return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

				return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
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

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil && len(response.Choices[0].Delta.ToolCalls) > 0 {
			completion += response.Choices[0].Delta.ToolCalls[0].Function.Arguments
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

		// 替换成调用的模型
		response.Model = mak.ReqModel.Model

		// OpenAI官方格式
		if len(response.ResponseBytes) > 0 {

			data := make(map[string]interface{})
			if err = gjson.Unmarshal(response.ResponseBytes, &data); err != nil {
				logger.Error(ctx, err)
				return err
			}

			// 替换成调用的模型
			if _, ok := data["model"]; ok {
				data["model"] = mak.ReqModel.Model
			}

			if err = util.SSEServer(ctx, gjson.MustEncodeString(data)); err != nil {
				logger.Error(ctx, err)
				return err
			}

		} else {
			if err = util.SSEServer(ctx, gjson.MustEncodeString(response)); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}
}

// 保存日志
func (s *sChat) SaveLog(ctx context.Context, reqModel, realModel *model.Model, modelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int) {

	if len(retry) == 0 {
		s.mutex.Lock()
		defer s.mutex.Unlock()
	}

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && (errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) || errors.Is(completionsRes.Error, errors.ERR_MODEL_DISABLED)) {
		return
	}

	chat := do.Chat{
		TraceId:          gctx.CtxId(ctx),
		UserId:           service.Session().GetUserId(ctx),
		AppId:            service.Session().GetAppId(ctx),
		IsSmartMatch:     isSmartMatch,
		Stream:           completionsReq.Stream,
		PromptTokens:     completionsRes.Usage.PromptTokens,
		CompletionTokens: completionsRes.Usage.CompletionTokens,
		TotalTokens:      completionsRes.Usage.TotalTokens,
		SearchTokens:     completionsRes.Usage.SearchTokens,
		CacheWriteTokens: completionsRes.Usage.CacheCreationInputTokens,
		CacheHitTokens:   completionsRes.Usage.CacheReadInputTokens,
		ConnTime:         completionsRes.ConnTime,
		Duration:         completionsRes.Duration,
		TotalTime:        completionsRes.TotalTime,
		InternalTime:     completionsRes.InternalTime,
		ReqTime:          completionsRes.EnterTime,
		ReqDate:          gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:         g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:         g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:          util.GetLocalIp(),
		Status:           1,
		Host:             g.RequestFromCtx(ctx).GetHost(),
	}

	if config.Cfg.Log.Open && len(completionsReq.Messages) > 0 && slices.Contains(config.Cfg.Log.Records, "prompt") {

		prompt := completionsReq.Messages[len(completionsReq.Messages)-1].Content

		if reqModel.Type == 102 {

			if multiContent, ok := prompt.([]interface{}); ok {
				for _, value := range multiContent {
					if content, ok := value.(map[string]interface{}); ok {
						if content["type"] == "text" {
							chat.Prompt = gconv.String(content["text"])
						}
					}
				}
			} else {
				chat.Prompt = gconv.String(prompt)
			}

		} else {

			if slices.Contains(config.Cfg.Log.Records, "image") {
				chat.Prompt = gconv.String(prompt)
			} else {
				if multiContent, ok := prompt.([]interface{}); ok {

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "image_url" {
								if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {
									if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {
										imageUrl["url"] = "[BASE64图像数据]"
									}
								}
							}

							if content["type"] == "image" {
								if source, ok := content["source"].(sdkm.Source); ok {
									source.Data = "[BASE64图像数据]"
									content["source"] = source
								}
							}
						}
					}

					chat.Prompt = gconv.String(multiContent)

				} else {
					chat.Prompt = gconv.String(prompt)
				}
			}
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.Records, "completion") {
		chat.Completion = completionsRes.Completion
	}

	if reqModel != nil {
		chat.Corp = reqModel.Corp
		chat.ModelId = reqModel.Id
		chat.Name = reqModel.Name
		chat.Model = reqModel.Model
		chat.Type = reqModel.Type
		chat.TextQuota = reqModel.TextQuota
		chat.MultimodalQuota = reqModel.MultimodalQuota

		if reqModel.Type == 102 {
			chat.TextQuota.BillingMethod = reqModel.MultimodalAudioQuota.AudioQuota.BillingMethod
			chat.TextQuota.PromptRatio = reqModel.MultimodalAudioQuota.AudioQuota.PromptRatio
			chat.TextQuota.CompletionRatio = reqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio
			chat.TextQuota.FixedQuota = reqModel.MultimodalAudioQuota.AudioQuota.FixedQuota
		}
	}

	if realModel != nil {
		chat.IsEnablePresetConfig = realModel.IsEnablePresetConfig
		chat.PresetConfig = realModel.PresetConfig
		chat.IsEnableForward = realModel.IsEnableForward
		chat.ForwardConfig = realModel.ForwardConfig
		chat.IsEnableModelAgent = realModel.IsEnableModelAgent
		chat.RealModelId = realModel.Id
		chat.RealModelName = realModel.Name
		chat.RealModel = realModel.Model
	}

	if chat.IsEnableModelAgent && modelAgent != nil {
		chat.ModelAgentId = modelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			Corp:    modelAgent.Corp,
			Name:    modelAgent.Name,
			BaseUrl: modelAgent.BaseUrl,
			Path:    modelAgent.Path,
			Weight:  modelAgent.Weight,
			Remark:  modelAgent.Remark,
			Status:  modelAgent.Status,
		}
	}

	if fallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = fallbackModel.Model
		chat.FallbackConfig.ModelName = fallbackModel.Name
	}

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
		if common.IsAborted(completionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.Records, "messages") {
		for _, message := range completionsReq.Messages {

			content := message.Content

			if !slices.Contains(config.Cfg.Log.Records, "image") {

				if multiContent, ok := content.([]interface{}); ok {

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "image_url" {
								if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {
									if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {
										imageUrl["url"] = "[BASE64图像数据]"
									}
								}
							}

							if content["type"] == "image" {
								if source, ok := content["source"].(sdkm.Source); ok {
									source.Data = "[BASE64图像数据]"
									content["source"] = source
								}
							}
						}
					}

					content = gconv.String(multiContent)
				}
			}

			chat.Messages = append(chat.Messages, mcommon.Message{
				Role:    message.Role,
				Content: gconv.String(content),
			})
		}
	}

	if retryInfo != nil {

		chat.IsRetry = retryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sChat SaveLog error: %v", err)

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sChat SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, modelAgent, fallbackModelAgent, fallbackModel, key, completionsReq, completionsRes, retryInfo, isSmartMatch, retry...)
	}
}
