package google

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/grand"
	"github.com/iimeta/fastapi-sdk/google"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/iimeta/tiktoken-go"
	"io"
	"math"
	"slices"
)

type sGoogle struct{}

func init() {
	service.RegisterGoogle(New())
}

func New() service.IGoogle {
	return &sGoogle{}
}

// Completions
func (s *sGoogle) Completions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGoogle Completions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convToChatCompletionRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      *google.Client
		res         sdkm.GoogleChatCompletionRes
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

					if response.Usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

				} else {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
				}

				body := make(map[string]interface{})
				if err := gjson.Unmarshal(request.GetBody(), &body); err == nil {
					if t, ok := body["tools"]; ok {
						if tools := gconv.String(t); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
							totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
						}
					}
				} else {
					logger.Error(ctx, err)
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

				if response.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
				}

				if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
				}

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

				if response.Usage.PromptTokensDetails.TextTokens > 0 {
					textTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
				}

				if response.Usage.PromptTokensDetails.AudioTokens > 0 {
					audioTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				} else {
					audioTokens = int(math.Ceil(float64(response.Usage.PromptTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				}

				if response.Usage.CompletionTokensDetails.TextTokens > 0 {
					textTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
				}

				if response.Usage.CompletionTokensDetails.AudioTokens > 0 {
					audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				} else {
					audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				}

				totalTokens = textTokens + audioTokens

				if response.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
				}

				if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
				}

			} else if mak.ReqModel.Type != 100 {
				if mak.ReqModel.TextQuota.BillingMethod == 1 {

					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

					if response.Usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
					}

					if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
					}

				} else {
					totalTokens = mak.ReqModel.TextQuota.FixedQuota
				}
			}
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 分组折扣
			if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
				totalTokens = int(math.Ceil(float64(totalTokens) * mak.Group.Discount))
			}

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

				service.Chat().SaveLog(ctx, model.ChatLog{
					Group:              mak.Group,
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					CompletionsReq:     &params,
					CompletionsRes:     completionsRes,
					RetryInfo:          retryInfo,
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

	if client, err = common.NewGoogleClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
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
func (s *sGoogle) CompletionsStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGoogle CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convToChatCompletionRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      *google.Client
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

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					body := make(map[string]interface{})
					if err := gjson.Unmarshal(request.GetBody(), &body); err == nil {
						if t, ok := body["tools"]; ok {
							if tools := gconv.String(t); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
								totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
							}
						}
					} else {
						logger.Error(ctx, err)
					}

				} else if mak.ReqModel.Type == 102 { // 多模态语音

					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
					}

				} else {
					if mak.ReqModel.TextQuota.BillingMethod == 1 {

						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

						if usage.PromptTokensDetails.CachedTokens != 0 {
							totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
						}

						if usage.CompletionTokensDetails.CachedTokens != 0 {
							totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
						}

					} else {
						usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
						totalTokens = mak.ReqModel.TextQuota.FixedQuota
					}
				}

			} else if retryInfo == nil && usage != nil && mak.ReqModel != nil {

				if mak.ReqModel.Type == 100 { // 多模态

					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					body := make(map[string]interface{})
					if err := gjson.Unmarshal(request.GetBody(), &body); err == nil {
						if t, ok := body["tools"]; ok {
							if tools := gconv.String(t); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
								totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
							}
						}
					} else {
						logger.Error(ctx, err)
					}

				} else if mak.ReqModel.Type == 102 { // 多模态语音

					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
					}

				} else {
					if mak.ReqModel.TextQuota.BillingMethod == 1 {

						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

						if usage.PromptTokensDetails.CachedTokens != 0 {
							totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
						}

						if usage.CompletionTokensDetails.CachedTokens != 0 {
							totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
						}

					} else {
						usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
						totalTokens = mak.ReqModel.TextQuota.FixedQuota
					}
				}
			}

			if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

				// 分组折扣
				if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
					totalTokens = int(math.Ceil(float64(totalTokens) * mak.Group.Discount))
				}

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

					service.Chat().SaveLog(ctx, model.ChatLog{
						Group:              mak.Group,
						ReqModel:           mak.ReqModel,
						RealModel:          mak.RealModel,
						ModelAgent:         mak.ModelAgent,
						FallbackModelAgent: fallbackModelAgent,
						FallbackModel:      fallbackModel,
						Key:                mak.Key,
						CompletionsReq:     &params,
						CompletionsRes:     completionsRes,
						RetryInfo:          retryInfo,
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

	if client, err = common.NewGoogleClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
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

	googleChatCompletionReq := sdkm.GoogleChatCompletionReq{}
	if err := gjson.Unmarshal(request.GetBody(), &googleChatCompletionReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return sdkm.ChatCompletionRequest{}
	}

	messages := make([]sdkm.ChatCompletionMessage, 0)
	for _, content := range googleChatCompletionReq.Contents {

		contents := make([]interface{}, 0)
		for _, part := range content.Parts {

			if part.Text != "" {
				contents = append(contents, g.MapStrAny{
					"type": "text",
					"text": part.Text,
				})
			}

			if part.InlineData != nil {
				contents = append(contents, g.MapStrAny{
					"type": "image_url",
					"image_url": g.MapStrAny{
						"url": part.InlineData.Data,
					},
				})
			}
		}

		role := content.Role

		if role == consts.ROLE_MODEL {
			role = consts.ROLE_ASSISTANT
		}

		messages = append(messages, sdkm.ChatCompletionMessage{
			Role:    role,
			Content: contents,
		})
	}

	return sdkm.ChatCompletionRequest{
		Model:       request.GetRouterMap()["model"],
		Messages:    messages,
		MaxTokens:   googleChatCompletionReq.GenerationConfig.MaxOutputTokens,
		Temperature: googleChatCompletionReq.GenerationConfig.Temperature,
		TopP:        googleChatCompletionReq.GenerationConfig.TopP,
	}
}

func convToChatCompletionResponse(ctx context.Context, res sdkm.GoogleChatCompletionRes, stream bool) sdkm.ChatCompletionResponse {

	googleChatCompletionRes := sdkm.GoogleChatCompletionRes{
		ResponseBytes: res.ResponseBytes,
		UsageMetadata: res.UsageMetadata,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &googleChatCompletionRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := sdkm.ChatCompletionResponse{
		ID:            consts.COMPLETION_ID_PREFIX + grand.S(29),
		Object:        consts.COMPLETION_OBJECT,
		Created:       gtime.Timestamp(),
		Model:         googleChatCompletionRes.ModelVersion,
		ResponseBytes: res.ResponseBytes,
		ConnTime:      res.ConnTime,
		Duration:      res.Duration,
		TotalTime:     res.TotalTime,
		Error:         googleChatCompletionRes.Err,
	}

	if len(googleChatCompletionRes.Candidates) > 0 {
		if stream {
			for _, candidate := range googleChatCompletionRes.Candidates {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
					Index: candidate.Index,
					Delta: &sdkm.ChatCompletionStreamChoiceDelta{
						Role:    consts.ROLE_ASSISTANT,
						Content: candidate.Content.Parts[0].Text,
					},
				})
			}
		} else {
			for i, part := range googleChatCompletionRes.Candidates[0].Content.Parts {
				chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
					Index: i,
					Message: &sdkm.ChatCompletionMessage{
						Role:    consts.ROLE_ASSISTANT,
						Content: part.Text,
					},
					FinishReason: "stop",
				})
			}
		}
	}

	if googleChatCompletionRes.UsageMetadata != nil {
		chatCompletionResponse.Usage = &sdkm.Usage{
			PromptTokens:     googleChatCompletionRes.UsageMetadata.PromptTokenCount,
			CompletionTokens: googleChatCompletionRes.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      googleChatCompletionRes.UsageMetadata.TotalTokenCount,
		}
	}

	return chatCompletionResponse
}
