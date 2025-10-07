package chat

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

// SmartCompletions
func (s *sChat) SmartCompletions(ctx context.Context, params smodel.ChatCompletionRequest, reqModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error) {

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
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

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

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			response.Usage = billingData.Usage
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
					completionsRes.Usage.TotalTokens = spend.TotalSpendTokens
				}

				if retryInfo == nil && len(response.Choices) > 0 && response.Choices[0].Message != nil {
					completionsRes.Completion = gconv.String(response.Choices[0].Message.Content)
				}

				s.SaveLog(ctx, model.ChatLog{
					ReqModel:           reqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					CompletionsReq:     &params,
					CompletionsRes:     completionsRes,
					RetryInfo:          retryInfo,
					Spend:              spend,
					IsSmartMatch:       true,
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

	params.Model = mak.RealModel.Model

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if params.Messages[0].Role == sconsts.ROLE_SYSTEM {
				params.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, params.Messages[1:]...)
			} else {
				params.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
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

	response, err = common.NewAdapter(ctx, mak, false).ChatCompletions(ctx, params)
	if err != nil {
		logger.Error(ctx, err)

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
