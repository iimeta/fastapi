package chat

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/tiktoken-go"
	"math"
)

// SmartCompletions
func (s *sChat) SmartCompletions(ctx context.Context, params sdkm.ChatCompletionRequest, reqModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SmartCompletions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              reqModel.Model,
			Messages:           params.Messages,
			ReqModel:           reqModel,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      sdk.Client
		retryInfo   *mcommon.Retry
		textTokens  int
		imageTokens int
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.RealModel != nil {

			model := mak.RealModel.Model
			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if mak.RealModel.Type == 100 { // 多模态
				if response.Usage == nil {

					response.Usage = new(sdkm.Usage)

					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, mak.RealModel)
						response.Usage.PromptTokens = textTokens + imageTokens
					} else {
						if response.Usage.PromptTokens == 0 {
							response.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
						}
					}

					if response.Usage.CompletionTokens == 0 && len(response.Choices) > 0 && response.Choices[0].Message != nil {
						response.Usage.CompletionTokens = common.GetCompletionTokens(ctx, model, gconv.String(response.Choices[0].Message.Content))
					}

					response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.RealModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.RealModel.MultimodalQuota.TextQuota.CompletionRatio))

				} else {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.RealModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.RealModel.MultimodalQuota.TextQuota.CompletionRatio))
				}

			} else if response.Usage == nil || response.Usage.TotalTokens == 0 {

				response.Usage = new(sdkm.Usage)

				response.Usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)

				if len(response.Choices) > 0 && response.Choices[0].Message != nil {
					response.Usage.CompletionTokens = common.GetCompletionTokens(ctx, model, gconv.String(response.Choices[0].Message.Content))
				}

				response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
			}
		}

		if mak.RealModel != nil && response.Usage != nil {
			if mak.RealModel.Type != 100 {
				if mak.RealModel.TextQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.RealModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*mak.RealModel.TextQuota.CompletionRatio))
				} else {
					totalTokens = mak.RealModel.TextQuota.FixedQuota
				}
			}
		}

		if mak.RealModel != nil {
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
					completionsRes.Completion = gconv.String(response.Choices[0].Message.Content)
				}

				s.SaveLog(ctx, reqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo, true)
			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	params.Model = mak.RealModel.Model

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if params.Messages[0].Role == consts.ROLE_SYSTEM {
				params.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, params.Messages[1:]...)
			} else {
				params.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, params.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if params.MaxTokens != 0 {
			if mak.RealModel.PresetConfig.MinTokens != 0 && params.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
				params.MaxTokens = mak.RealModel.PresetConfig.MinTokens
			} else if mak.RealModel.PresetConfig.MaxTokens != 0 && params.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
				params.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
			}
		}
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sChat SmartCompletions params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])
				params.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.ChatCompletion(ctx, params)
	if err != nil {
		logger.Error(ctx, err)

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
							return s.SmartCompletions(g.RequestFromCtx(ctx).GetCtx(), params, reqModel, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(g.RequestFromCtx(ctx).GetCtx(), params, reqModel, nil, fallbackModel)
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

			return s.SmartCompletions(g.RequestFromCtx(ctx).GetCtx(), params, reqModel, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}
