package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	sdk "github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"math"
)

// SmartCompletions
func (s *sChat) SmartCompletions(ctx context.Context, params sdkm.ChatCompletionRequest, reqModel *model.Model, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SmartCompletions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		client     sdk.Chat
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
		key        string
		baseUrl    string
		path       string
		agentTotal int
		keyTotal   int
		retryInfo  *do.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && err == nil {

			if response.Usage == nil || response.Usage.TotalTokens == 0 {

				response.Usage = new(sdkm.Usage)
				model := reqModel.Model

				if common.GetCorpCode(ctx, reqModel.Corp) != consts.CORP_OPENAI {
					model = consts.DEFAULT_MODEL
				} else {
					if _, err := tiktoken.EncodingForModel(model); err != nil {
						model = consts.DEFAULT_MODEL
					}
				}

				promptTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(model, params.Messages); err != nil {
					logger.Errorf(ctx, "sChat SmartCompletions model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					response.Usage.PromptTokens = promptTokens
					logger.Debugf(ctx, "sChat SmartCompletions NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-promptTime)
				}

				if len(response.Choices) > 0 {
					completionTime := gtime.TimestampMilli()
					if completionTokens, err := tiktoken.NumTokensFromString(model, response.Choices[0].Message.Content); err != nil {
						logger.Errorf(ctx, "sChat SmartCompletions model: %s, completion: %s, NumTokensFromString error: %v", params.Model, response.Choices[0].Message.Content, err)
					} else {
						response.Usage.CompletionTokens = completionTokens
						logger.Debugf(ctx, "sChat SmartCompletions NumTokensFromString len(completion): %d, time: %d", len(response.Choices[0].Message.Content), gtime.TimestampMilli()-completionTime)
					}
				}
			}

			if reqModel != nil {
				// 替换成调用的模型
				response.Model = reqModel.Model
				// 实际消费额度
				if reqModel.BillingMethod == 1 {
					response.Usage.TotalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.CompletionRatio))
				} else {
					response.Usage.TotalTokens = reqModel.FixedQuota
				}
			}
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

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
			}

			if retryInfo == nil && len(response.Choices) > 0 && response.Choices[0].Message != nil {
				completionsRes.Completion = response.Choices[0].Message.Content
			}

			s.SaveChat(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo, true)

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
	}

	if realModel.IsEnableForward {
		if realModel, err = service.Model().GetTargetModel(ctx, realModel, params.Messages[len(params.Messages)-1].Content); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	baseUrl = realModel.BaseUrl
	path = realModel.Path

	if realModel.IsEnableModelAgent {

		if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {
				if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
					retryInfo = &do.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.SmartCompletions(ctx, params, reqModel, fallbackModel)
				}
			}

			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl
			path = modelAgent.Path

			if keyTotal, k, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				logger.Error(ctx, err)

				service.ModelAgent().RecordErrorModelAgent(ctx, realModel, modelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent)
				}

				if realModel.IsEnableFallback {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &do.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.SmartCompletions(ctx, params, reqModel, fallbackModel)
					}
				}

				return response, err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {
				if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
					retryInfo = &do.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.SmartCompletions(ctx, params, reqModel, fallbackModel)
				}
			}

			return response, err
		}
	}

	if k == nil {
		return response, errors.ERR_NO_AVAILABLE_KEY
	}

	request := params
	request.Model = realModel.Model
	key = k.Key

	if common.GetCorpCode(ctx, realModel.Corp) == consts.CORP_BAIDU {
		key = getAccessToken(ctx, k.Key, baseUrl, config.Cfg.Http.ProxyUrl)
	}

	// 替换预设提示词
	if reqModel.Prompt != "" {
		if request.Messages[0].Role == consts.ROLE_SYSTEM {
			request.Messages = append([]sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: reqModel.Prompt,
			}}, request.Messages[1:]...)
		} else {
			request.Messages = append([]sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: reqModel.Prompt,
			}}, request.Messages...)
		}
	}

	client, err = common.NewClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &do.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.SmartCompletions(ctx, params, reqModel, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.ChatCompletion(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if realModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, k)
				} else {
					service.Key().DisabledModelKey(ctx, k)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(realModel.IsEnableModelAgent, agentTotal, keyTotal, len(retry)) {
				if realModel.IsEnableFallback {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &do.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.SmartCompletions(ctx, params, reqModel, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &do.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.SmartCompletions(ctx, params, reqModel, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}
