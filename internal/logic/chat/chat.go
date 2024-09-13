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
	"time"
)

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

// Completions
func (s *sChat) Completions(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

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
		client      sdk.Client
		reqModel    *model.Model
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

			// 替换成调用的模型
			response.Model = reqModel.Model
			model := reqModel.Model

			if !tiktoken.IsEncodingForModel(model) {
				model = consts.DEFAULT_MODEL
			}

			if reqModel.Type == 100 { // 多模态
				if response.Usage == nil {

					response.Usage = new(sdkm.Usage)

					if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
						textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, reqModel)
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
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*reqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*reqModel.MultimodalQuota.TextQuota.CompletionRatio))

				} else {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*reqModel.MultimodalQuota.TextQuota.CompletionRatio))
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

		if reqModel != nil && response.Usage != nil {
			if reqModel.Type != 100 {
				if reqModel.TextQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
				} else {
					totalTokens = reqModel.TextQuota.FixedQuota
				}
			}
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
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

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo, false)

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

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

	if realModel.IsEnableModelAgent {
		if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {
				if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
					retryInfo = &mcommon.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.Completions(ctx, params, fallbackModel)
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
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent, "No available model agent key")
				}

				if realModel.IsEnableFallback {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.Completions(ctx, params, fallbackModel)
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
					retryInfo = &mcommon.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.Completions(ctx, params, fallbackModel)
				}
			}

			return response, err
		}
	}

	request := params
	key = k.Key

	if !gstr.Contains(realModel.Model, "*") {
		request.Model = realModel.Model
	}

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
							return s.Completions(ctx, params, fallbackModel)
						}
					}
					return response, err
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				return s.Completions(ctx, params, fallbackModel, append(retry, 1)...)
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
			if request.Messages[0].Role == consts.ROLE_SYSTEM {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: realModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages[1:]...)
			} else {
				request.Messages = append([]sdkm.ChatCompletionMessage{{
					Role:    consts.ROLE_SYSTEM,
					Content: realModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if request.MaxTokens != 0 {
			if realModel.PresetConfig.MinTokens != 0 && request.MaxTokens < realModel.PresetConfig.MinTokens {
				request.MaxTokens = realModel.PresetConfig.MinTokens
			} else if realModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > realModel.PresetConfig.MaxTokens {
				request.MaxTokens = realModel.PresetConfig.MaxTokens
			}
		}
	}

	client, err = common.NewClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.Completions(ctx, params, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.ChatCompletion(ctx, request)
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
						return s.Completions(ctx, params, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Completions(ctx, params, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// CompletionsStream
func (s *sChat) CompletionsStream(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModel *model.Model, retry ...int) (err error) {

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
		client      sdk.Client
		reqModel    *model.Model
		realModel   = new(model.Model)
		k           *model.Key
		modelAgent  *model.ModelAgent
		key         string
		baseUrl     string
		path        string
		completion  string
		agentTotal  int
		keyTotal    int
		connTime    int64
		duration    int64
		totalTime   int64
		textTokens  int
		imageTokens int
		totalTokens int
		usage       *sdkm.Usage
		retryInfo   *mcommon.Retry
		projectId   string
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			if retryInfo == nil && completion != "" && (usage == nil || usage.PromptTokens == 0 || usage.CompletionTokens == 0) {

				if usage == nil {
					usage = new(sdkm.Usage)
				}

				model := reqModel.Model
				if !tiktoken.IsEncodingForModel(model) {
					model = consts.DEFAULT_MODEL
				}

				if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
					textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, reqModel)
					usage.PromptTokens = textTokens + imageTokens
				} else {
					if usage.PromptTokens == 0 {
						usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
					}
				}

				if usage.CompletionTokens == 0 {
					usage.CompletionTokens = common.GetCompletionTokens(ctx, model, completion)
				}

				if reqModel.Type == 100 { // 多模态
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*reqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*reqModel.MultimodalQuota.TextQuota.CompletionRatio))
				} else {
					if reqModel.TextQuota.BillingMethod == 1 {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
					} else {
						usage.TotalTokens = reqModel.TextQuota.FixedQuota
						totalTokens = reqModel.TextQuota.FixedQuota
					}
				}

			} else if retryInfo == nil && usage != nil {

				if reqModel.Type == 100 { // 多模态
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*reqModel.MultimodalQuota.TextQuota.CompletionRatio))
				} else {
					if reqModel.TextQuota.BillingMethod == 1 {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						totalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
					} else {
						usage.TotalTokens = reqModel.TextQuota.FixedQuota
						totalTokens = reqModel.TextQuota.FixedQuota
					}
				}
			}

			if retryInfo == nil && (err == nil || common.IsAborted(err)) {
				if err := grpool.Add(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.Add(ctx, func(ctx context.Context) {

				realModel.ModelAgent = modelAgent

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

				s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo, false)

			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
	}

	if realModel.IsEnableForward {
		if realModel, err = service.Model().GetTargetModel(ctx, realModel, params.Messages); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	baseUrl = realModel.BaseUrl
	path = realModel.Path

	if realModel.IsEnableModelAgent {
		if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {
				if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
					retryInfo = &mcommon.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.CompletionsStream(ctx, params, fallbackModel)
				}
			}

			return err
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
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.CompletionsStream(ctx, params, fallbackModel)
					}
				}

				return err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, realModel); err != nil {
			logger.Error(ctx, err)

			if realModel.IsEnableFallback {
				if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
					retryInfo = &mcommon.Retry{
						IsRetry:    true,
						RetryCount: len(retry),
						ErrMsg:     err.Error(),
					}
					return s.CompletionsStream(ctx, params, fallbackModel)
				}
			}

			return err
		}
	}

	request := params
	key = k.Key

	if !gstr.Contains(realModel.Model, "*") {
		request.Model = realModel.Model
	}

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
							return s.CompletionsStream(ctx, params, fallbackModel)
						}
					}
					return err
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				return s.CompletionsStream(ctx, params, fallbackModel, append(retry, 1)...)
			}

			return err
		}

		path = fmt.Sprintf(path, projectId, realModel.Model)

	} else if common.GetCorpCode(ctx, realModel.Corp) == consts.CORP_BAIDU {
		key = getBaiduToken(ctx, k.Key, baseUrl, config.Cfg.Http.ProxyUrl)
	}

	// 替换预设提示词
	if reqModel.IsEnablePresetConfig && reqModel.PresetConfig.IsSupportSystemRole && reqModel.PresetConfig.SystemRolePrompt != "" {
		if request.Messages[0].Role == consts.ROLE_SYSTEM {
			request.Messages = append([]sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: reqModel.PresetConfig.SystemRolePrompt,
			}}, request.Messages[1:]...)
		} else {
			request.Messages = append([]sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: reqModel.PresetConfig.SystemRolePrompt,
			}}, request.Messages...)
		}
	}

	client, err = common.NewClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.CompletionsStream(ctx, params, fallbackModel)
			}
		}

		return err
	}

	response, err := client.ChatCompletionStream(ctx, request)
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
						return s.CompletionsStream(ctx, params, fallbackModel)
					}
				}
				return err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.CompletionsStream(ctx, params, fallbackModel, append(retry, 1)...)
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
						if response.Usage.CompletionTokensDetails.ReasoningTokens != 0 {
							usage.CompletionTokensDetails.ReasoningTokens = response.Usage.CompletionTokensDetails.ReasoningTokens
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
							return s.CompletionsStream(ctx, params, fallbackModel)
						}
					}
					return err
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				return s.CompletionsStream(ctx, params, fallbackModel, append(retry, 1)...)
			}

			return err
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil {
			completion += response.Choices[0].Delta.Content
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
			}
		}

		// 替换成调用的模型
		response.Model = reqModel.Model

		// OpenAI官方格式
		if len(response.ResponseBytes) > 0 {

			data := make(map[string]interface{})
			if err = gjson.Unmarshal(response.ResponseBytes, &data); err != nil {
				logger.Error(ctx, err)
				return err
			}

			// 替换成调用的模型
			if _, ok := data["model"]; ok {
				data["model"] = reqModel.Model
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
func (s *sChat) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && (errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) || errors.Is(completionsRes.Error, errors.ERR_MODEL_DISABLED)) {
		return
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: isSmartMatch,
		Stream:       completionsReq.Stream,
		ConnTime:     completionsRes.ConnTime,
		Duration:     completionsRes.Duration,
		TotalTime:    completionsRes.TotalTime,
		InternalTime: completionsRes.InternalTime,
		ReqTime:      completionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
	}

	if slices.Contains(config.Cfg.RecordLogs, "prompt") {
		chat.Prompt = gconv.String(completionsReq.Messages[len(completionsReq.Messages)-1].Content)
	}

	if slices.Contains(config.Cfg.RecordLogs, "completion") {
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

		if chat.IsEnableModelAgent && realModel.ModelAgent != nil {
			chat.ModelAgentId = realModel.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
				Corp:    realModel.ModelAgent.Corp,
				Name:    realModel.ModelAgent.Name,
				BaseUrl: realModel.ModelAgent.BaseUrl,
				Path:    realModel.ModelAgent.Path,
				Weight:  realModel.ModelAgent.Weight,
				Remark:  realModel.ModelAgent.Remark,
				Status:  realModel.ModelAgent.Status,
			}
		}
	}

	chat.PromptTokens = completionsRes.Usage.PromptTokens
	chat.CompletionTokens = completionsRes.Usage.CompletionTokens
	chat.TotalTokens = completionsRes.Usage.TotalTokens

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
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

	if slices.Contains(config.Cfg.RecordLogs, "messages") {
		for _, message := range completionsReq.Messages {
			chat.Messages = append(chat.Messages, mcommon.Message{
				Role:    message.Role,
				Content: gconv.String(message.Content),
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
		logger.Error(ctx, err)

		if len(retry) == 5 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sChat SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModel, key, completionsReq, completionsRes, retryInfo, isSmartMatch, retry...)
	}
}
