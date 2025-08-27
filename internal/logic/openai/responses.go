package openai

import (
	"context"
	"fmt"
	"io"
	"math"
	"slices"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
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
)

type sOpenAI struct{}

func init() {
	service.RegisterOpenAI(New())
}

func New() service.IOpenAI {
	return &sOpenAI{}
}

// Responses
func (s *sOpenAI) Responses(ctx context.Context, request *ghttp.Request, isChatCompletions bool, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.OpenAIResponsesRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sOpenAI Responses time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = common.ConvResponsesToChatCompletionsRequest(request, isChatCompletions)
		mak    = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo   *mcommon.Retry
		textTokens  int
		imageTokens int
		audioTokens int
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		chatCompletionResponse := common.ConvResponsesToChatCompletionsResponse(ctx, response)

		if isChatCompletions {
			response.ResponseBytes = gjson.MustEncode(chatCompletionResponse)
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				chatCompletionResponse.Model = mak.ReqModel.Model
			}

			model := mak.ReqModel.Model

			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if mak.ReqModel.Type == 100 { // 多模态

				if chatCompletionResponse.Usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

					chatCompletionResponse.Usage = new(sdkm.Usage)

					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, mak.ReqModel)
						chatCompletionResponse.Usage.PromptTokens = textTokens + imageTokens
					} else {
						if chatCompletionResponse.Usage.PromptTokens == 0 {
							chatCompletionResponse.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
						}
					}

					if chatCompletionResponse.Usage.CompletionTokens == 0 && len(chatCompletionResponse.Choices) > 0 && chatCompletionResponse.Choices[0].Message != nil {
						for _, choice := range chatCompletionResponse.Choices {
							chatCompletionResponse.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
						}
					}

					chatCompletionResponse.Usage.TotalTokens = chatCompletionResponse.Usage.PromptTokens + chatCompletionResponse.Usage.CompletionTokens
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

				} else {
					totalTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
				}

				if params.Tools != nil {
					if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
						totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
						chatCompletionResponse.Usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
					}
				}

				if params.WebSearchOptions != nil {
					searchTokens := common.GetMultimodalSearchTokens(ctx, params.WebSearchOptions, mak.ReqModel)
					totalTokens += searchTokens
					chatCompletionResponse.Usage.SearchTokens = searchTokens
				}

				if chatCompletionResponse.Usage.CacheCreationInputTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
				}

				if chatCompletionResponse.Usage.CacheReadInputTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
				}

				if chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
				}

				if chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
				}

			} else if mak.ReqModel.Type == 102 { // 多模态语音

				if chatCompletionResponse.Usage == nil {

					chatCompletionResponse.Usage = new(sdkm.Usage)

					textTokens, audioTokens = common.GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
					chatCompletionResponse.Usage.PromptTokens = textTokens + audioTokens

					if len(chatCompletionResponse.Choices) > 0 && chatCompletionResponse.Choices[0].Message != nil && chatCompletionResponse.Choices[0].Message.Audio != nil {
						for _, choice := range chatCompletionResponse.Choices {
							chatCompletionResponse.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, choice.Message.Audio.Transcript) + 388
						}
					}
				}

				chatCompletionResponse.Usage.TotalTokens = chatCompletionResponse.Usage.PromptTokens + chatCompletionResponse.Usage.CompletionTokens
				totalTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

				if chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
				}

				if chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
				}

			} else if chatCompletionResponse.Usage == nil || chatCompletionResponse.Usage.TotalTokens == 0 {

				chatCompletionResponse.Usage = new(sdkm.Usage)

				chatCompletionResponse.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)

				if len(chatCompletionResponse.Choices) > 0 && chatCompletionResponse.Choices[0].Message != nil {
					for _, choice := range chatCompletionResponse.Choices {
						chatCompletionResponse.Usage.CompletionTokens += common.GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
					}
				}

				chatCompletionResponse.Usage.TotalTokens = chatCompletionResponse.Usage.PromptTokens + chatCompletionResponse.Usage.CompletionTokens
			}
		}

		if mak.ReqModel != nil && chatCompletionResponse.Usage != nil {
			if mak.ReqModel.Type == 102 {

				if chatCompletionResponse.Usage.PromptTokensDetails.TextTokens > 0 {
					textTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
				}

				if chatCompletionResponse.Usage.PromptTokensDetails.AudioTokens > 0 {
					audioTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				} else {
					audioTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
				}

				if chatCompletionResponse.Usage.CompletionTokensDetails.TextTokens > 0 {
					textTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
				}

				if chatCompletionResponse.Usage.CompletionTokensDetails.AudioTokens > 0 {
					audioTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				} else {
					audioTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
				}

				totalTokens = textTokens + audioTokens

				if chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
				}

				if chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
				}

			} else if mak.ReqModel.Type != 100 {
				if mak.ReqModel.TextQuota.BillingMethod == 1 {

					totalTokens = int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(chatCompletionResponse.Usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

					if chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
					}

					if chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
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
					ConnTime:     chatCompletionResponse.ConnTime,
					Duration:     chatCompletionResponse.Duration,
					TotalTime:    chatCompletionResponse.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && chatCompletionResponse.Usage != nil {

					if chatCompletionResponse.Usage.PromptTokensDetails.CachedTokens != 0 {
						chatCompletionResponse.Usage.CacheCreationInputTokens = response.Usage.PromptTokensDetails.CachedTokens
					}

					if chatCompletionResponse.Usage.CompletionTokensDetails.CachedTokens != 0 {
						chatCompletionResponse.Usage.CacheReadInputTokens = response.Usage.CompletionTokensDetails.CachedTokens
					}

					completionsRes.Usage = *chatCompletionResponse.Usage
					completionsRes.Usage.TotalTokens = totalTokens
				}

				if retryInfo == nil && len(chatCompletionResponse.Choices) > 0 && chatCompletionResponse.Choices[0].Message != nil {
					if mak.RealModel.Type == 102 && chatCompletionResponse.Choices[0].Message.Audio != nil {
						completionsRes.Completion = chatCompletionResponse.Choices[0].Message.Audio.Transcript
					} else {
						if len(chatCompletionResponse.Choices) > 1 {
							for i, choice := range chatCompletionResponse.Choices {

								if choice.Message.Content != nil {
									completionsRes.Completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, gconv.String(choice.Message.Content))
								}

								if choice.Message.ToolCalls != nil {
									completionsRes.Completion += fmt.Sprintf("index: %d\ntool_calls: %s\n\n", i, gconv.String(choice.Message.ToolCalls))
								}
							}
						} else {

							if chatCompletionResponse.Choices[0].Message.ReasoningContent != nil {
								completionsRes.Completion = gconv.String(chatCompletionResponse.Choices[0].Message.ReasoningContent)
							}

							completionsRes.Completion += gconv.String(chatCompletionResponse.Choices[0].Message.Content)

							if chatCompletionResponse.Choices[0].Message.ToolCalls != nil {
								completionsRes.Completion += fmt.Sprintf("\ntool_calls: %s", gconv.String(chatCompletionResponse.Choices[0].Message.ToolCalls))
							}
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

	data := request.GetBody()

	if isChatCompletions {
		data = gjson.MustEncode(common.ConvChatCompletionsToResponsesRequest(request))
	}

	response, err = common.NewOpenAIAdapter(ctx, mak, false).Responses(ctx, data)
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

					if params.WebSearchOptions != nil {
						searchTokens := common.GetMultimodalSearchTokens(ctx, params.WebSearchOptions, mak.ReqModel)
						totalTokens += searchTokens
						usage.SearchTokens = searchTokens
					}

					if usage.CacheCreationInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
					}

					if usage.CacheReadInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
					}

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
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

					if params.Tools != nil {
						if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
							totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
							usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
						}
					}

					if params.WebSearchOptions != nil {
						searchTokens := common.GetMultimodalSearchTokens(ctx, params.WebSearchOptions, mak.ReqModel)
						totalTokens += searchTokens
						usage.SearchTokens = searchTokens
					}

					if usage.CacheCreationInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
					}

					if usage.CacheReadInputTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
					}

					if usage.PromptTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
					}

					if usage.CompletionTokensDetails.CachedTokens != 0 {
						totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
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

						if usage.PromptTokensDetails.CachedTokens != 0 {
							usage.CacheCreationInputTokens = usage.PromptTokensDetails.CachedTokens
						}

						if usage.CompletionTokensDetails.CachedTokens != 0 {
							usage.CacheReadInputTokens = usage.CompletionTokensDetails.CachedTokens
						}

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

	data := request.GetBody()

	if isChatCompletions {
		data = gjson.MustEncode(common.ConvChatCompletionsToResponsesRequest(request))
	}

	response, err := common.NewOpenAIAdapter(ctx, mak, true).ResponsesStream(ctx, data)
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
		//if mak.ReqModel.IsEnableForward {
		//	response.Model = mak.ReqModel.Model
		//}

		// OpenAI官方格式
		//if len(res.ResponseBytes) > 0 {
		//
		//	data := make(map[string]interface{})
		//	if err = gjson.Unmarshal(res.ResponseBytes, &data); err != nil {
		//		logger.Error(ctx, err)
		//		return err
		//	}
		//
		//	// 替换成调用的模型
		//	if mak.ReqModel.IsEnableForward {
		//		if _, ok := data["model"]; ok {
		//			data["model"] = mak.ReqModel.Model
		//		}
		//	}
		//
		//	if err = util.SSEServer(ctx, gjson.MustEncodeString(data)); err != nil {
		//		logger.Error(ctx, err)
		//		return err
		//	}
		//
		//} else {

		data := res.ResponseBytes
		if isChatCompletions {
			data = gjson.MustEncode(response)
		}

		if err = util.SSEServer(ctx, string(data)); err != nil {
			logger.Error(ctx, err)
			return err
		}
		//}
	}
}
