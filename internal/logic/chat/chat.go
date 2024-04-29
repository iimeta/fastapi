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
	"github.com/sashabaranov/go-openai"
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

	var reqModel *model.Model
	var realModel = new(model.Model)
	var k *model.Key
	var modelAgent *model.ModelAgent
	var key string
	var baseUrl string
	var path string
	var agentTotal int
	var keyTotal int
	var isRetry bool

	defer func() {

		// 不记录重试
		if isRetry {
			return
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if err == nil {

			if response.Usage == nil || response.Usage.TotalTokens == 0 {

				response.Usage = new(openai.Usage)
				model := reqModel.Model

				if reqModel.Corp != consts.CORP_OPENAI {
					model = "gpt-3.5-turbo"
				} else {
					if _, err := tiktoken.EncodingForModel(model); err != nil {
						model = "gpt-3.5-turbo"
					}
				}

				numTokensFromMessagesTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(model, params.Messages); err != nil {
					logger.Errorf(ctx, "sChat Completions model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					logger.Debugf(ctx, "sChat NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-numTokensFromMessagesTime)
					response.Usage.PromptTokens = promptTokens
				}

				if len(response.Choices) > 0 {
					completionTime := gtime.TimestampMilli()
					if completionTokens, err := tiktoken.NumTokensFromString(model, response.Choices[0].Message.Content); err != nil {
						logger.Errorf(ctx, "sChat Completions model: %s, completion: %s, NumTokensFromString error: %v", params.Model, response.Choices[0].Message.Content, err)
					} else {
						logger.Debugf(ctx, "sChat NumTokensFromString len(completion): %d, time: %d", len(response.Choices[0].Message.Content), gtime.TimestampMilli()-completionTime)
						response.Usage.CompletionTokens = completionTokens
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

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, reqModel, *response.Usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				reqModel.ModelAgent = modelAgent

				completionsRes := &model.CompletionsRes{
					Error:        err,
					ConnTime:     response.ConnTime,
					Duration:     response.Duration,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if response.Usage != nil {
					completionsRes.Usage = *response.Usage
				}

				if len(response.Choices) > 0 {
					completionsRes.Completion = response.Choices[0].Message.Content
				}

				s.SaveChat(ctx, reqModel, realModel, k, &params, completionsRes)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

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

				service.ModelAgent().RecordErrorModelAgent(ctx, realModel, modelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent)
				}

				logger.Error(ctx, err)

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

	if realModel.Corp == consts.CORP_BAIDU {
		key = getAccessToken(ctx, k.Key, baseUrl, config.Cfg.Http.ProxyUrl)
	}

	// 替换预设提示词
	if reqModel.Prompt != "" {
		if request.Messages[0].Role == openai.ChatMessageRoleSystem {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: reqModel.Prompt,
			}}, request.Messages[1:]...)
		} else {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: reqModel.Prompt,
			}}, request.Messages...)
		}
	}

	client := sdk.NewClient(ctx, realModel.Corp, realModel.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	if response, err = client.ChatCompletion(ctx, request); err != nil {

		logger.Error(ctx, err)

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

		apiError := &sdkerr.APIError{}
		if errors.As(err, &apiError) {

			isRetry = true
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			switch apiError.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return response, err
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
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

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
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

				response, err = s.Completions(ctx, params, append(retry, 1)...)
			}

			return response, err
		}

		reqError := &sdkerr.RequestError{}
		if errors.As(err, &reqError) {

			isRetry = true
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			switch reqError.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return response, err
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
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

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
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

				response, err = s.Completions(ctx, params, append(retry, 1)...)
			}

			return response, err
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

	var reqModel *model.Model
	var realModel = new(model.Model)
	var k *model.Key
	var modelAgent *model.ModelAgent
	var key string
	var baseUrl string
	var path string
	var completion string
	var agentTotal int
	var keyTotal int
	var connTime int64
	var duration int64
	var totalTime int64
	var isRetry bool
	var usage *openai.Usage

	defer func() {

		// 不记录重试
		if isRetry {
			return
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if completion != "" && usage == nil {

				usage = new(openai.Usage)
				model := reqModel.Model

				if reqModel.Corp != consts.CORP_OPENAI {
					model = "gpt-3.5-turbo"
				} else {
					if _, err := tiktoken.EncodingForModel(model); err != nil {
						model = "gpt-3.5-turbo"
					}
				}

				numTokensFromMessagesTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(model, params.Messages); err != nil {
					logger.Errorf(ctx, "sChat CompletionsStream model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					usage.PromptTokens = promptTokens
				}
				logger.Debugf(ctx, "sChat NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-numTokensFromMessagesTime)

				completionTime := gtime.TimestampMilli()
				if completionTokens, err := tiktoken.NumTokensFromString(model, completion); err != nil {
					logger.Errorf(ctx, "sChat CompletionsStream model: %s, completion: %s, NumTokensFromString error: %v", params.Model, completion, err)
				} else {
					logger.Debugf(ctx, "sChat NumTokensFromString len(completion): %d, time: %d", len(completion), gtime.TimestampMilli()-completionTime)

					usage.CompletionTokens = completionTokens

					if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
						if err := service.Common().RecordUsage(ctx, reqModel, *usage); err != nil {
							logger.Error(ctx, err)
						}
					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}

				// 实际消费额度
				usage.TotalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.PromptRatio + float64(usage.CompletionTokens)*reqModel.CompletionRatio))
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				reqModel.ModelAgent = modelAgent

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

				s.SaveChat(ctx, reqModel, realModel, k, &params, completionsRes)
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

				service.ModelAgent().RecordErrorModelAgent(ctx, realModel, modelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent)
				}

				logger.Error(ctx, err)

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

	if realModel.Corp == consts.CORP_BAIDU {
		key = getAccessToken(ctx, k.Key, baseUrl, config.Cfg.Http.ProxyUrl)
	}

	// 替换预设提示词
	if reqModel.Prompt != "" {
		if request.Messages[0].Role == openai.ChatMessageRoleSystem {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: reqModel.Prompt,
			}}, request.Messages[1:]...)
		} else {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: reqModel.Prompt,
			}}, request.Messages...)
		}
	}

	client := sdk.NewClient(ctx, realModel.Corp, realModel.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	response, err := client.ChatCompletionStream(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

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

		apiError := &sdkerr.APIError{}
		if errors.As(err, &apiError) {

			isRetry = true
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			switch apiError.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return err
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
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

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
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

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			}

			return err
		}

		reqError := &sdkerr.RequestError{}
		if errors.As(err, &reqError) {

			isRetry = true
			service.Common().RecordError(ctx, realModel, k, modelAgent)

			switch reqError.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return err
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
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

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
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

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)
			}

			return err
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		if response == nil || response.Error != nil {
			return response.Error
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

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {

			if err = util.SSEServer(ctx, "", gjson.MustEncode(response)); err != nil {
				logger.Error(ctx, err)
				return err
			}

			if err = util.SSEServer(ctx, "", "[DONE]"); err != nil {
				logger.Error(ctx, err)
				return err
			}

			return nil
		}

		if err = util.SSEServer(ctx, "", gjson.MustEncode(response)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

// 保存文生文聊天数据
func (s *sChat) SaveChat(ctx context.Context, model *model.Model, realModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
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
		chat.IsEnableModelAgent = model.IsEnableModelAgent
		chat.IsForward = model.IsForward

		if chat.IsEnableModelAgent && model.ModelAgent != nil {
			chat.ModelAgentId = model.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
				Name:    model.ModelAgent.Name,
				BaseUrl: model.ModelAgent.BaseUrl,
				Path:    model.ModelAgent.Path,
				Weight:  model.ModelAgent.Weight,
				Remark:  model.ModelAgent.Remark,
				Status:  model.ModelAgent.Status,
			}
		}

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

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
		if errors.Is(completionsRes.Error, context.Canceled) ||
			gstr.Contains(chat.ErrMsg, "broken pipe") ||
			gstr.Contains(chat.ErrMsg, "aborted") {
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

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
