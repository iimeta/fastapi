package audio

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gorilla/websocket"
	sdk "github.com/iimeta/fastapi-sdk"
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
)

type sRealtime struct {
	upgrader websocket.Upgrader
}

func init() {
	service.RegisterRealtime(New())
}

func New() service.IRealtime {
	return &sRealtime{}
}

// Realtime
func (s *sRealtime) Realtime(ctx context.Context, r *ghttp.Request, params model.RealtimeRequest, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sRealtime Realtime time: %d", gtime.TimestampMilli()-now)
	}()

	conn, err := s.upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if conn != nil {
		defer func() {
			if err := conn.Close(); err != nil {
				logger.Error(ctx, err)
			}
		}()
	}

	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	var (
		client      *sdk.RealtimeClient
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

				//if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
				//	textTokens, imageTokens = common.GetMultimodalTokens(ctx, model, content, reqModel)
				//	usage.PromptTokens = textTokens + imageTokens
				//} else {
				if usage.PromptTokens == 0 {
					usage.PromptTokens = common.GetPromptTokens(ctx, model, params.Messages)
				}
				//}

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

				//s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo, false)

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
					return s.Realtime(ctx, r, params, fallbackModel)
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
						return s.Realtime(ctx, r, params, fallbackModel)
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
					return s.Realtime(ctx, r, params, fallbackModel)
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

	client, err = common.NewRealtimeClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.Realtime(ctx, r, params, fallbackModel)
			}
		}

		return err
	}

	var (
		requestChan = make(chan *sdkm.RealtimeRequest)
		messageType = websocket.TextMessage
		message     []byte
	)

	defer close(requestChan)

	_ = grpool.Add(ctx, func(ctx context.Context) {
		requestChan <- &sdkm.RealtimeRequest{
			MessageType: messageType,
		}
	})

	response, err := client.Realtime(ctx, requestChan)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, realModel, k, modelAgent)

		_, isDisabled := common.IsNeedRetry(err)

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

		// todo
		logger.Debug(ctx, agentTotal, keyTotal)

		//if isRetry {
		//	if common.IsMaxRetry(realModel.IsEnableModelAgent, agentTotal, keyTotal, len(retry)) {
		//		if realModel.IsEnableFallback {
		//			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
		//				retryInfo = &mcommon.Retry{
		//					IsRetry:    true,
		//					RetryCount: len(retry),
		//					ErrMsg:     err.Error(),
		//				}
		//				return s.Realtime(ctx, r, params, fallbackModel)
		//			}
		//		}
		//		return err
		//	}
		//
		//	retryInfo = &mcommon.Retry{
		//		IsRetry:    true,
		//		RetryCount: len(retry),
		//		ErrMsg:     err.Error(),
		//	}
		//
		//	return s.Realtime(ctx, r, params, fallbackModel, append(retry, 1)...)
		//}

		return err
	}

	if err := grpool.Add(ctx, func(ctx context.Context) {

		defer close(response)

		for {

			response := <-response

			if response == nil {
				return
			}

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
						return
					}

					return
				}

				err = response.Error

				// 记录错误次数和禁用
				service.Common().RecordError(ctx, realModel, k, modelAgent)

				_, isDisabled := common.IsNeedRetry(err)

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

				//if isRetry {
				//	if common.IsMaxRetry(realModel.IsEnableModelAgent, agentTotal, keyTotal, len(retry)) {
				//		if realModel.IsEnableFallback {
				//			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				//				retryInfo = &mcommon.Retry{
				//					IsRetry:    true,
				//					RetryCount: len(retry),
				//					ErrMsg:     err.Error(),
				//				}
				//				return s.Realtime(ctx, r, params, fallbackModel)
				//			}
				//		}
				//		return
				//	}
				//
				//	retryInfo = &mcommon.Retry{
				//		IsRetry:    true,
				//		RetryCount: len(retry),
				//		ErrMsg:     err.Error(),
				//	}
				//
				//	return s.Realtime(ctx, r, params, fallbackModel, append(retry, 1)...)
				//}

				return
			}

			//if len(response.Choices) > 0 && response.Choices[0].Delta != nil {
			//	completion += response.Choices[0].Delta.Content
			//}
			//
			//if len(response.Choices) > 0 && response.Choices[0].Delta != nil && len(response.Choices[0].Delta.ToolCalls) > 0 {
			//	completion += response.Choices[0].Delta.ToolCalls[0].Function.Arguments
			//}

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
			//response.Model = reqModel.Model

			// OpenAI官方格式
			if len(response.Message) > 0 {
				//if err = conn.WriteJSON(response.Message); err != nil {
				if err = conn.WriteMessage(messageType, response.Message); err != nil {
					logger.Error(ctx, err)
					return
				}
			}
		}

	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	for {

		messageType, message, err = conn.ReadMessage()
		if err != nil {

			requestChan <- nil

			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}

			logger.Error(ctx, err)
			return err
		}

		requestChan <- &sdkm.RealtimeRequest{
			MessageType: messageType,
			Message:     message,
		}
	}
}
