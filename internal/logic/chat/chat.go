package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"math"
)

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

// Completions
func (s *sChat) Completions(ctx context.Context, params sdkm.ChatCompletionRequest, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat Completions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		reqModel   *model.Model
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

				if getCorpCode(ctx, reqModel.Corp) != consts.CORP_OPENAI {
					model = consts.DEFAULT_MODEL
				} else {
					if _, err := tiktoken.EncodingForModel(model); err != nil {
						model = consts.DEFAULT_MODEL
					}
				}

				promptTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(model, params.Messages); err != nil {
					logger.Errorf(ctx, "sChat Completions model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					response.Usage.PromptTokens = promptTokens
					logger.Debugf(ctx, "sChat Completions NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-promptTime)
				}

				if len(response.Choices) > 0 {
					completionTime := gtime.TimestampMilli()
					if completionTokens, err := tiktoken.NumTokensFromString(model, response.Choices[0].Message.Content); err != nil {
						logger.Errorf(ctx, "sChat Completions model: %s, completion: %s, NumTokensFromString error: %v", params.Model, response.Choices[0].Message.Content, err)
					} else {
						response.Usage.CompletionTokens = completionTokens
						logger.Debugf(ctx, "sChat Completions NumTokensFromString len(completion): %d, time: %d", len(response.Choices[0].Message.Content), gtime.TimestampMilli()-completionTime)
					}
				}
			}

			if reqModel != nil {
				// 替换成调用的模型
				response.Model = reqModel.Model
				// 实际消费额度
				response.Usage.TotalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.CompletionRatio))
			}
		}

		if retryInfo == nil && (err == nil || isAborted(err)) {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, reqModel, response.Usage); err != nil {
					logger.Error(ctx, err)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
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

			s.SaveChat(ctx, reqModel, realModel, k, &params, completionsRes, retryInfo)

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	*realModel = *reqModel

	if realModel.IsForward {
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

				return response, err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, realModel); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	request := params
	request.Model = realModel.Model
	key = k.Key

	if getCorpCode(ctx, realModel.Corp) == consts.CORP_BAIDU {
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

	client := sdk.NewClient(ctx, getCorpCode(ctx, realModel.Corp), realModel.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	response, err = client.ChatCompletion(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, realModel, k, modelAgent)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return response, err
			} else if config.Cfg.Api.Retry < 0 {
				if realModel.IsEnableModelAgent {
					if len(retry) == agentTotal {
						return response, err
					}
				} else if len(retry) == keyTotal {
					return response, err
				}
			} else if config.Cfg.Api.Retry == 0 {
				return response, err
			}
		}

		isRetry, isDisabled := isNeedRetry(err)

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
			retryInfo = &do.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}
			return s.Completions(ctx, params, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// CompletionsStream
func (s *sChat) CompletionsStream(ctx context.Context, params sdkm.ChatCompletionRequest, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		reqModel   *model.Model
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
		key        string
		baseUrl    string
		path       string
		completion string
		agentTotal int
		keyTotal   int
		connTime   int64
		duration   int64
		totalTime  int64
		usage      *sdkm.Usage
		retryInfo  *do.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if retryInfo == nil && (completion != "" && usage == nil) {

				usage = new(sdkm.Usage)
				model := reqModel.Model

				if getCorpCode(ctx, reqModel.Corp) != consts.CORP_OPENAI {
					model = consts.DEFAULT_MODEL
				} else {
					if _, err := tiktoken.EncodingForModel(model); err != nil {
						model = consts.DEFAULT_MODEL
					}
				}

				promptTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(model, params.Messages); err != nil {
					logger.Errorf(ctx, "sChat CompletionsStream model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					usage.PromptTokens = promptTokens
					logger.Debugf(ctx, "sChat CompletionsStream NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-promptTime)
				}

				completionTime := gtime.TimestampMilli()
				if completionTokens, err := tiktoken.NumTokensFromString(model, completion); err != nil {
					logger.Errorf(ctx, "sChat CompletionsStream model: %s, completion: %s, NumTokensFromString error: %v", params.Model, completion, err)
				} else {
					usage.CompletionTokens = completionTokens
					logger.Debugf(ctx, "sChat CompletionsStream NumTokensFromString len(completion): %d, time: %d", len(completion), gtime.TimestampMilli()-completionTime)
				}

				// 实际消费额度
				usage.TotalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.PromptRatio + float64(usage.CompletionTokens)*reqModel.CompletionRatio))
			}

			if retryInfo == nil && (err == nil || isAborted(err)) {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, reqModel, usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

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
				}

				s.SaveChat(ctx, reqModel, realModel, k, &params, completionsRes, retryInfo)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return err
	}

	*realModel = *reqModel

	if realModel.IsForward {
		if realModel, err = service.Model().GetTargetModel(ctx, realModel, params.Messages[len(params.Messages)-1].Content); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	baseUrl = realModel.BaseUrl
	path = realModel.Path

	if realModel.IsEnableModelAgent {

		if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, realModel); err != nil {
			logger.Error(ctx, err)
			return err
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

				return err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, realModel); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	request := params
	request.Model = realModel.Model
	key = k.Key

	if getCorpCode(ctx, realModel.Corp) == consts.CORP_BAIDU {
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

	client := sdk.NewClient(ctx, getCorpCode(ctx, realModel.Corp), realModel.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	response, err := client.ChatCompletionStream(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, realModel, k, modelAgent)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return err
			} else if config.Cfg.Api.Retry < 0 {
				if realModel.IsEnableModelAgent {
					if len(retry) == agentTotal {
						return err
					}
				} else if len(retry) == keyTotal {
					return err
				}
			} else if config.Cfg.Api.Retry == 0 {
				return err
			}
		}

		isRetry, isDisabled := isNeedRetry(err)

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
			retryInfo = &do.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}
			return s.CompletionsStream(ctx, params, append(retry, 1)...)
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

			err = response.Error

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			isRetry, isDisabled := isNeedRetry(err)

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
				retryInfo = &do.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.CompletionsStream(ctx, params, append(retry, 1)...)
			}

			return err
		}

		if len(response.Choices) > 0 {
			completion += response.Choices[0].Delta.Content
		}

		if response.Usage != nil {
			// 实际消费额度
			response.Usage.TotalTokens = int(math.Ceil(reqModel.PromptRatio*float64(response.Usage.PromptTokens) + reqModel.CompletionRatio*float64(response.Usage.CompletionTokens)))
			usage = response.Usage
		}

		// 替换成调用的模型
		response.Model = reqModel.Model

		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {

			if err = util.SSEServer(ctx, gjson.MustEncode(response)); err != nil {
				logger.Error(ctx, err)
				return err
			}

			if err = util.SSEServer(ctx, "[DONE]"); err != nil {
				logger.Error(ctx, err)
				return err
			}

			return nil
		}

		if err = util.SSEServer(ctx, gjson.MustEncode(response)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

// 保存文生文聊天数据
func (s *sChat) SaveChat(ctx context.Context, model *model.Model, realModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *do.Retry, isSmartMatch ...bool) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: len(isSmartMatch) > 0 && isSmartMatch[0],
		Stream:       completionsReq.Stream,
		Prompt:       completionsReq.Messages[len(completionsReq.Messages)-1].Content,
		Completion:   completionsRes.Completion,
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

	if model != nil {

		chat.Corp = model.Corp
		chat.ModelId = model.Id
		chat.Name = model.Name
		chat.Model = model.Model
		chat.Type = model.Type
		chat.BillingMethod = model.BillingMethod
		chat.PromptRatio = model.PromptRatio
		chat.CompletionRatio = model.CompletionRatio
		chat.FixedQuota = model.FixedQuota
		chat.IsForward = model.IsForward

		if chat.IsForward && model.ForwardConfig != nil {

			chat.ForwardConfig = &do.ForwardConfig{
				ForwardRule:   model.ForwardConfig.ForwardRule,
				MatchRule:     model.ForwardConfig.MatchRule,
				TargetModel:   model.ForwardConfig.TargetModel,
				DecisionModel: model.ForwardConfig.DecisionModel,
				Keywords:      model.ForwardConfig.Keywords,
				TargetModels:  model.ForwardConfig.TargetModels,
			}

			chat.RealModelId = realModel.Id
			chat.RealModelName = realModel.Name
			chat.RealModel = realModel.Model
		}

		chat.PromptTokens = completionsRes.Usage.PromptTokens
		chat.CompletionTokens = completionsRes.Usage.CompletionTokens

		if model.BillingMethod == 1 {
			chat.TotalTokens = completionsRes.Usage.TotalTokens
		} else {
			chat.TotalTokens = chat.FixedQuota
		}
	}

	if realModel.IsEnableModelAgent && realModel.ModelAgent != nil {
		chat.IsEnableModelAgent = realModel.IsEnableModelAgent
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

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
		if isAborted(completionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	for _, message := range completionsReq.Messages {
		chat.Messages = append(chat.Messages, do.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	if retryInfo != nil {

		chat.IsRetry = retryInfo.IsRetry
		chat.Retry = &do.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if chat.IsRetry && completionsRes.Error == nil {
			chat.Status = 3
			chat.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}

func getCorpCode(ctx context.Context, corpId string) string {

	corp, err := service.Corp().GetCacheCorp(ctx, corpId)
	if err != nil || corp == nil {
		corp, err = service.Corp().GetCorpAndSaveCache(ctx, corpId)
	}

	if corp != nil {
		return corp.Code
	}

	return corpId
}

func isAborted(err error) bool {
	return errors.Is(err, context.Canceled) ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}

func isNeedRetry(err error) (isRetry bool, isDisabled bool) {

	apiError := &sdkerr.ApiError{}
	if errors.As(err, &apiError) {

		switch apiError.HttpStatusCode {
		case 400:
			if errors.Is(err, sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED) {
				return false, false
			}
		case 401, 429:
			if errors.Is(err, sdkerr.ERR_INVALID_API_KEY) || errors.Is(err, sdkerr.ERR_INSUFFICIENT_QUOTA) {
				return true, true
			}
		}

		return true, false
	}

	reqError := &sdkerr.RequestError{}
	if errors.As(err, &reqError) {
		return true, false
	}

	return false, false
}
