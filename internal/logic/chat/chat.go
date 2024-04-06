package chat

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/sashabaranov/go-openai"
	"strings"
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
func (s *sChat) Completions(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (response sdkm.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat Completions time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl string
	var keyTotal int
	var isRetry bool

	defer func() {

		// 不记录重试
		if isRetry {
			return
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, m, response.Usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				completionsRes := &model.CompletionsRes{
					Usage:        response.Usage,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if len(response.Choices) > 0 {
					completionsRes.Completion = response.Choices[0].Message.Content
				}

				s.SaveChat(ctx, m, key, &params, completionsRes)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if m.IsEnableModelAgent {

		if modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if keyTotal, key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, m, modelAgent)
				logger.Error(ctx, err)
				return response, err
			}
		}

	} else {
		if keyTotal, key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	request := params
	request.Model = m.Model

	if gstr.HasPrefix(m.Model, "glm-") {

		tmp := new(model.Key)
		*tmp = *key
		key = tmp

		key.Key = genGlmSign(ctx, key.Key)

		if request.TopP == 1 {
			request.TopP -= 0.01
		} else if request.TopP == 0 {
			request.TopP += 0.01
		}

		if request.Temperature == 1 {
			request.Temperature -= 0.01
		} else if request.Temperature == 0 {
			request.Temperature += 0.01
		}

		if request.Messages[0].Role == openai.ChatMessageRoleSystem && request.Messages[0].Content == "" && len(request.Messages[0].ToolCalls) == 0 {
			request.Messages = request.Messages[1:]
		}
	}

	// 替换预设提示词
	if m.Prompt != "" {
		if request.Messages[0].Role == openai.ChatMessageRoleSystem {
			request.Messages[0].Content = m.Prompt
		} else {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: m.Prompt,
			}}, request.Messages...)
		}
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, baseUrl)
	if response, err = sdk.ChatCompletion(ctx, client, request); err != nil {
		logger.Error(ctx, err)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return response, err
			} else if config.Cfg.Api.Retry < 0 && len(retry) == keyTotal {
				return response, err
			} else if config.Cfg.Api.Retry == 0 {
				return response, err
			}
		}

		e := &openai.APIError{}
		if errors.As(err, &e) {

			isRetry = true
			service.Common().RecordError(ctx, m, key, modelAgent)

			switch e.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return response, err
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
						}

					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}

				response, err = s.Completions(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
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
func (s *sChat) CompletionsStream(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl string
	var completion string
	var keyTotal int
	var connTime int64
	var duration int64
	var totalTime int64
	var isRetry bool

	defer func() {

		// 不记录重试
		if isRetry {
			return
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			usage := openai.Usage{}

			if completion != "" {

				numTokensFromMessagesTime := gtime.TimestampMilli()
				if promptTokens, err := tiktoken.NumTokensFromMessages(m.Model, params.Messages); err != nil {
					logger.Errorf(ctx, "CompletionsStream model: %s, messages: %s, NumTokensFromMessages error: %v", params.Model, gjson.MustEncodeString(params.Messages), err)
				} else {
					usage.PromptTokens = promptTokens
				}
				logger.Debugf(ctx, "NumTokensFromMessages len(params.Messages): %d, time: %d", len(params.Messages), gtime.TimestampMilli()-numTokensFromMessagesTime)

				completionTime := gtime.TimestampMilli()
				if completionTokens, err := tiktoken.NumTokensFromString(m.Model, completion); err != nil {
					logger.Errorf(ctx, "CompletionsStream model: %s, completion: %s, NumTokensFromString error: %v", params.Model, completion, err)
				} else {
					logger.Debugf(ctx, "NumTokensFromString len(completion): %d, time: %d", len(completion), gtime.TimestampMilli()-completionTime)

					usage.CompletionTokens = completionTokens

					if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
						if err := service.Common().RecordUsage(ctx, m, usage); err != nil {
							logger.Error(ctx, err)
						}
					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
				m.ModelAgent = modelAgent
				s.SaveChat(ctx, m, key, &params, &model.CompletionsRes{
					Completion:   completion,
					Usage:        usage,
					Error:        err,
					ConnTime:     connTime,
					Duration:     duration,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				})
			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if m.IsEnableModelAgent {

		if modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if keyTotal, key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, m, modelAgent)
				logger.Error(ctx, err)
				return err
			}
		}

	} else {

		if keyTotal, key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	request := params
	request.Model = m.Model

	if gstr.HasPrefix(m.Model, "glm-") {

		tmp := new(model.Key)
		*tmp = *key
		key = tmp

		key.Key = genGlmSign(ctx, key.Key)

		if request.TopP == 1 {
			request.TopP -= 0.01
		} else if request.TopP == 0 {
			request.TopP += 0.01
		}

		if request.Temperature == 1 {
			request.Temperature -= 0.01
		} else if request.Temperature == 0 {
			request.Temperature += 0.01
		}

		if request.Messages[0].Role == openai.ChatMessageRoleSystem && request.Messages[0].Content == "" && len(request.Messages[0].ToolCalls) == 0 {
			request.Messages = request.Messages[1:]
		}
	}

	// 替换预设提示词
	if m.Prompt != "" {
		if request.Messages[0].Role == openai.ChatMessageRoleSystem {
			request.Messages[0].Content = m.Prompt
		} else {
			request.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: m.Prompt,
			}}, request.Messages...)
		}
	}

	client := sdk.NewClient(ctx, m.Model, key.Key, baseUrl)
	response, err := sdk.ChatCompletionStream(ctx, client, request)
	if err != nil {
		logger.Error(ctx, err)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return err
			} else if config.Cfg.Api.Retry < 0 && len(retry) == keyTotal {
				return err
			} else if config.Cfg.Api.Retry == 0 {
				return err
			}
		}

		e := &openai.APIError{}
		if errors.As(err, &e) {

			isRetry = true
			service.Common().RecordError(ctx, m, key, modelAgent)

			switch e.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return err
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
						}

					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}

				err = s.CompletionsStream(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
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

		if response == nil {
			return nil
		}

		completion += response.Choices[0].Delta.Content
		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Choices[0].FinishReason == "stop" {

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
func (s *sChat) SaveChat(ctx context.Context, model *model.Model, key *model.Key, completionsReq *openai.ChatCompletionRequest, completionsRes *model.CompletionsRes) {

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
	}

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Usage.TotalTokens != 0 {
		chat.PromptTokens = int(chat.PromptRatio * float64(completionsRes.Usage.PromptTokens))
		chat.CompletionTokens = int(chat.CompletionRatio * float64(completionsRes.Usage.CompletionTokens))
		chat.TotalTokens = chat.PromptTokens + chat.CompletionTokens
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
		chat.Status = -1
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

func genGlmSign(ctx context.Context, key string) string {

	split := strings.Split(key, ".")
	if len(split) != 2 {
		return key
	}

	now := gtime.Now()

	claims := jwt.MapClaims{
		"api_key":   split[0],
		"exp":       now.Add(time.Minute * 10).UnixMilli(),
		"timestamp": now.UnixMilli(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	token.Header["alg"] = "HS256"
	token.Header["sign_type"] = "SIGN"

	sign, err := token.SignedString([]byte(split[1]))
	if err != nil {
		logger.Error(ctx, err)
	}

	return sign
}
