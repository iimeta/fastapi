package chat

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
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
		client      sdk.Client
		realModel   = new(model.Model)
		k           *model.Key
		modelAgent  *model.ModelAgent
		key         string
		baseUrl     string
		path        string
		agentTotal  int
		keyTotal    int
		retryInfo   *mcommon.Retry
		textTokens  int
		imageTokens int
		totalTokens int
		projectId   string
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {

			model := realModel.Model
			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if realModel.Type == 100 { // 多模态
				if response.Usage == nil {

					response.Usage = new(sdkm.Usage)

					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, realModel)
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
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*realModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*realModel.MultimodalQuota.TextQuota.CompletionRatio))

				} else {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*realModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*realModel.MultimodalQuota.TextQuota.CompletionRatio))
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

		if realModel != nil && response.Usage != nil {
			if realModel.Type != 100 {
				if realModel.TextQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*realModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*realModel.TextQuota.CompletionRatio))
				} else {
					totalTokens = realModel.TextQuota.FixedQuota
				}
			}
		}

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

			realModel.ModelAgent = modelAgent

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

			s.SaveLog(ctx, reqModel, realModel, fallbackModelAgent, fallbackModel, k, &params, completionsRes, retryInfo, true)

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
	}

	if realModel.IsEnableForward {
		if realModel, err = service.Model().GetTargetModel(ctx, realModel, params.Messages); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	baseUrl = realModel.BaseUrl
	path = realModel.Path

	if fallbackModelAgent != nil || realModel.IsEnableModelAgent {

		if fallbackModelAgent != nil {
			modelAgent = fallbackModelAgent
		} else {

			if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, realModel); err != nil {
				logger.Error(ctx, err)

				if realModel.IsEnableFallback {

					if realModel.FallbackConfig.ModelAgent != "" {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, realModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel)
						}
					}

					if realModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(ctx, params, reqModel, nil, fallbackModel)
						}
					}
				}

				return response, err
			}
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl
			path = modelAgent.Path

			if keyTotal, k, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				logger.Error(ctx, err)

				service.ModelAgent().RecordErrorModelAgent(ctx, realModel, modelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent, "No available model agent key")
				}

				if realModel.IsEnableFallback {

					if realModel.FallbackConfig.ModelAgent != "" && realModel.FallbackConfig.ModelAgent != modelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, realModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel)
						}
					}

					if realModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(ctx, params, reqModel, nil, fallbackModel)
						}
					}
				}

				return response, err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {

				if realModel.FallbackConfig.ModelAgent != "" {
					if fallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, realModel); fallbackModelAgent != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel)
					}
				}

				if realModel.FallbackConfig.Model != "" {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.SmartCompletions(ctx, params, reqModel, nil, fallbackModel)
					}
				}
			}

			return response, err
		}
	}

	params.Model = realModel.Model
	key = k.Key

	if common.GetCorpCode(ctx, realModel.Corp) == consts.CORP_GCP_CLAUDE {

		projectId, key, err = getGcpTokenNew(ctx, k, config.Cfg.Http.ProxyUrl)
		if err != nil {
			logger.Error(ctx, err)

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			isRetry, isDisabled := common.IsNeedRetry(err)

			if isDisabled {
				if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
					if realModel.IsEnableModelAgent {
						service.ModelAgent().DisabledModelAgentKey(ctx, k, err.Error())
					} else {
						service.Key().DisabledModelKey(ctx, k, err.Error())
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if isRetry {
				if common.IsMaxRetry(realModel.IsEnableModelAgent, agentTotal, keyTotal, len(retry)) {
					if realModel.IsEnableFallback {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel)
						}
					}
					return response, err
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel, append(retry, 1)...)
			}

			return response, err
		}

		path = fmt.Sprintf(path, projectId, realModel.Model)

	} else if common.GetCorpCode(ctx, realModel.Corp) == consts.CORP_BAIDU {
		key = getBaiduToken(ctx, k.Key, baseUrl, config.Cfg.Http.ProxyUrl)
	}

	// 预设配置
	if realModel.IsEnablePresetConfig {

		// 替换预设提示词
		if realModel.PresetConfig.IsSupportSystemRole && realModel.PresetConfig.SystemRolePrompt != "" {
			if params.Messages[0].Role == consts.ROLE_SYSTEM {
				params.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: realModel.PresetConfig.SystemRolePrompt,
				}}, params.Messages[1:]...)
			} else {
				params.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: realModel.PresetConfig.SystemRolePrompt,
				}}, params.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if params.MaxTokens != 0 {
			if realModel.PresetConfig.MinTokens != 0 && params.MaxTokens < realModel.PresetConfig.MinTokens {
				params.MaxTokens = realModel.PresetConfig.MinTokens
			} else if realModel.PresetConfig.MaxTokens != 0 && params.MaxTokens > realModel.PresetConfig.MaxTokens {
				params.MaxTokens = realModel.PresetConfig.MaxTokens
			}
		}
	}

	if client, err = common.NewClient(ctx, realModel, key, baseUrl, path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.ChatCompletion(ctx, params)
	if err != nil {
		logger.Error(ctx, err)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if realModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, k, err.Error())
				} else {
					service.Key().DisabledModelKey(ctx, k, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {
			if common.IsMaxRetry(realModel.IsEnableModelAgent, agentTotal, keyTotal, len(retry)) {
				if realModel.IsEnableFallback {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.SmartCompletions(ctx, params, reqModel, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}
