package realtime

import (
	"context"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	serrors "github.com/iimeta/fastapi-sdk/errors"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sRealtime struct {
	upgrader websocket.Upgrader
}

func init() {
	service.RegisterRealtime(New())
}

func New() service.IRealtime {
	return &sRealtime{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 60 * time.Second,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			EnableCompression: true,
		},
	}
}

// Realtime
func (s *sRealtime) Realtime(ctx context.Context, r *ghttp.Request, params model.RealtimeRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sRealtime Realtime time: %d", gtime.TimestampMilli()-now)
	}()

	header := http.Header{
		"Trace-Id": {gctx.CtxId(ctx)},
	}

	if swp := r.Header.Values("Sec-Websocket-Protocol"); len(swp) > 0 && gstr.Contains(swp[0], "realtime") {
		header["Sec-Websocket-Protocol"] = []string{"realtime"}
	}

	conn, err := s.upgrader.Upgrade(r.Response.Writer, r.Request, header)
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
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		connTime  int64
		duration  int64
		totalTime int64
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err != nil && mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				completionsRes := &model.CompletionsRes{
					Error:        err,
					ConnTime:     connTime,
					Duration:     duration,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				spend := mcommon.Spend{}

				if mak.Group != nil {
					spend.GroupId = mak.Group.Id
					spend.GroupName = mak.Group.Name
					spend.GroupDiscount = mak.Group.Discount
				}

				s.SaveLog(ctx, model.ChatLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					CompletionsReq:     &smodel.ChatCompletionRequest{Stream: true},
					CompletionsRes:     completionsRes,
					RetryInfo:          retryInfo,
					Spend:              spend,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	requestChan := make(chan *smodel.RealtimeRequest)

	response, err := common.NewRealtimeAdapter(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path).Realtime(ctx, requestChan)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

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
							return s.Realtime(g.RequestFromCtx(ctx).GetCtx(), r, params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Realtime(g.RequestFromCtx(ctx).GetCtx(), r, params, nil, fallbackModel)
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

			return s.Realtime(g.RequestFromCtx(ctx).GetCtx(), r, params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

		defer close(response)

		responseMessage := ""
		responseCompletion := ""

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
					if err := conn.Close(); err != nil {
						logger.Error(ctx, err)
					}
					return
				}

				// 记录错误次数和禁用
				service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
					internalTime := gtime.TimestampMilli() - enterTime - totalTime

					completionsRes := &model.CompletionsRes{
						Error:        response.Error,
						ConnTime:     connTime,
						Duration:     duration,
						TotalTime:    totalTime,
						InternalTime: internalTime,
						EnterTime:    enterTime,
					}

					spend := mcommon.Spend{}

					if mak.Group != nil {
						spend.GroupId = mak.Group.Id
						spend.GroupName = mak.Group.Name
						spend.GroupDiscount = mak.Group.Discount
					}

					s.SaveLog(ctx, model.ChatLog{
						ReqModel:           mak.ReqModel,
						RealModel:          mak.RealModel,
						ModelAgent:         mak.ModelAgent,
						FallbackModelAgent: fallbackModelAgent,
						FallbackModel:      fallbackModel,
						Key:                mak.Key,
						CompletionsReq:     &smodel.ChatCompletionRequest{Stream: true},
						CompletionsRes:     completionsRes,
						RetryInfo:          retryInfo,
						Spend:              spend,
					})

				}); err != nil {
					logger.Error(ctx, err)
				}

				if err := conn.Close(); err != nil {
					logger.Error(ctx, err)
				}

				return
			}

			logger.Debugf(ctx, "sRealtime Response messageType: %d, message: %s", response.MessageType, response.Message)

			realtimeResponse := new(model.RealtimeResponse)
			if err = gjson.Unmarshal(response.Message, &realtimeResponse); err != nil {
				logger.Errorf(ctx, "sRealtime response.Message: %s, error: %v", response.Message, err)
				return
			}

			switch realtimeResponse.Type {
			case "conversation.item.input_audio_transcription.completed":
				if realtimeResponse.Transcript != "" {
					responseMessage = realtimeResponse.Transcript
				}
			case "response.audio_transcript.delta":
				responseCompletion += realtimeResponse.Delta
			case "response.text.done":
				if realtimeResponse.Text != "" {
					responseCompletion = realtimeResponse.Text
				}
			case "response.audio_transcript.done":
				if realtimeResponse.Transcript != "" {
					responseCompletion = realtimeResponse.Transcript
				}
			case "response.content_part.done":
				if realtimeResponse.Part.Text != "" {
					responseCompletion = realtimeResponse.Part.Text
				}
				if realtimeResponse.Part.Transcript != "" {
					responseCompletion = realtimeResponse.Part.Transcript
				}
			case "response.output_item.done":
				if len(realtimeResponse.Item.Content) > 0 {
					if realtimeResponse.Item.Content[0].Text != "" {
						responseCompletion = realtimeResponse.Item.Content[0].Text
					}
					if realtimeResponse.Item.Content[0].Transcript != "" {
						responseCompletion = realtimeResponse.Item.Content[0].Transcript
					}
				} else if realtimeResponse.Item.Arguments != nil {
					responseCompletion = gconv.String(realtimeResponse.Item.Arguments)
				}
			}

			if realtimeResponse.Response.Usage.TotalTokens != 0 {

				usage := &smodel.Usage{
					PromptTokens: realtimeResponse.Response.Usage.InputTokens,
					PromptTokensDetails: smodel.PromptTokensDetails{
						TextTokens:   realtimeResponse.Response.Usage.InputTokenDetails.TextTokens,
						AudioTokens:  realtimeResponse.Response.Usage.InputTokenDetails.AudioTokens,
						CachedTokens: realtimeResponse.Response.Usage.InputTokenDetails.CachedTokens,
					},
					CompletionTokens: realtimeResponse.Response.Usage.OutputTokens,
					CompletionTokensDetails: smodel.CompletionTokensDetails{
						TextTokens:  realtimeResponse.Response.Usage.OutputTokenDetails.TextTokens,
						AudioTokens: realtimeResponse.Response.Usage.OutputTokenDetails.AudioTokens,
					},
					TotalTokens: realtimeResponse.Response.Usage.TotalTokens,
				}

				message := responseMessage
				completion := responseCompletion

				typ := ""
				if len(realtimeResponse.Response.Output) > 0 {
					if len(realtimeResponse.Response.Output[0].Content) > 0 {
						typ = realtimeResponse.Response.Output[0].Content[0].Type
					} else {
						typ = realtimeResponse.Response.Output[0].Type
					}
				}

				billingData := &mcommon.BillingData{
					ChatCompletionRequest: smodel.ChatCompletionRequest{Stream: true, Messages: []smodel.ChatCompletionMessage{{Content: message}}},
					Completion:            completion,
					Usage:                 usage,
				}

				// 计算花费
				spend := common.Billing(ctx, mak, billingData)
				usage = billingData.Usage

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					if err := service.Common().RecordSpend(ctx, spend.TotalSpendTokens, mak.Key.Key, mak.Group); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
					internalTime := gtime.TimestampMilli() - enterTime - totalTime

					completionsRes := &model.CompletionsRes{
						Type:         typ,
						Completion:   completion,
						Error:        err,
						ConnTime:     response.ConnTime,
						Duration:     response.Duration,
						TotalTime:    response.TotalTime,
						InternalTime: internalTime,
						EnterTime:    enterTime,
					}

					completionsRes.Usage = *usage
					completionsRes.Usage.TotalTokens = spend.TotalSpendTokens

					s.SaveLog(ctx, model.ChatLog{
						ReqModel:           mak.ReqModel,
						RealModel:          mak.RealModel,
						ModelAgent:         mak.ModelAgent,
						FallbackModelAgent: fallbackModelAgent,
						FallbackModel:      fallbackModel,
						Key:                mak.Key,
						CompletionsReq:     &smodel.ChatCompletionRequest{Stream: true, Messages: []smodel.ChatCompletionMessage{{Content: message}}},
						CompletionsRes:     completionsRes,
						RetryInfo:          retryInfo,
						Spend:              spend,
					})

				}); err != nil {
					logger.Error(ctx, err)
				}

				responseMessage = ""
				responseCompletion = ""
			}

			if len(response.Message) > 0 {
				if err = conn.WriteMessage(response.MessageType, response.Message); err != nil {
					logger.Error(ctx, err)
					return
				}
			}

			if realtimeResponse.Error.Code != "" {
				if realtimeResponse.Error.Code == "session_expired" {
					if err := conn.Close(); err != nil {
						logger.Error(ctx, err)
					}
					return
				}
				logger.Error(ctx, realtimeResponse.Error)
			}
		}

	}, nil); err != nil {
		logger.Error(ctx, err)
		return err
	}

	defer close(requestChan)

	for {

		messageType, message, err := conn.ReadMessage()
		if err != nil {

			requestChan <- nil

			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				return nil
			}

			logger.Error(ctx, err)
			return err
		}

		logger.Debugf(ctx, "sRealtime Request messageType: %d, message: %s", messageType, message)

		if err := service.Auth().VerifySecretKey(ctx, service.Session().GetSecretKey(ctx)); err != nil {
			logger.Error(ctx, err)
			requestChan <- nil
			return err
		}

		requestChan <- &smodel.RealtimeRequest{
			MessageType: messageType,
			Message:     message,
		}
	}
}

// 保存日志
func (s *sRealtime) SaveLog(ctx context.Context, chatLog model.ChatLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sRealtime SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if chatLog.CompletionsRes.Error != nil && (errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: chatLog.IsSmartMatch,
		Stream:       chatLog.CompletionsReq.Stream,
		Spend:        chatLog.Spend,
		ConnTime:     chatLog.CompletionsRes.ConnTime,
		Duration:     chatLog.CompletionsRes.Duration,
		TotalTime:    chatLog.CompletionsRes.TotalTime,
		InternalTime: chatLog.CompletionsRes.InternalTime,
		ReqTime:      chatLog.CompletionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(chatLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if config.Cfg.Log.Open && len(chatLog.CompletionsReq.Messages) > 0 && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {
		chat.Prompt = gconv.String(chatLog.CompletionsReq.Messages[len(chatLog.CompletionsReq.Messages)-1].Content)
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") {
		chat.Completion = chatLog.CompletionsRes.Completion
	}

	if chatLog.ReqModel != nil {
		chat.ProviderId = chatLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, chatLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			chat.ProviderName = provider.Name
		}
		chat.ModelId = chatLog.ReqModel.Id
		chat.ModelName = chatLog.ReqModel.Name
		chat.Model = chatLog.ReqModel.Model
		chat.ModelType = chatLog.ReqModel.Type
	}

	if chatLog.RealModel != nil {
		chat.IsEnablePresetConfig = chatLog.RealModel.IsEnablePresetConfig
		chat.PresetConfig = chatLog.RealModel.PresetConfig
		chat.IsEnableForward = chatLog.RealModel.IsEnableForward
		chat.ForwardConfig = chatLog.RealModel.ForwardConfig
		chat.IsEnableModelAgent = chatLog.RealModel.IsEnableModelAgent
		chat.RealModelId = chatLog.RealModel.Id
		chat.RealModelName = chatLog.RealModel.Name
		chat.RealModel = chatLog.RealModel.Model
	}

	if chatLog.ModelAgent != nil {
		chat.IsEnableModelAgent = true
		chat.ModelAgentId = chatLog.ModelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			ProviderId: chatLog.ModelAgent.ProviderId,
			Name:       chatLog.ModelAgent.Name,
			BaseUrl:    chatLog.ModelAgent.BaseUrl,
			Path:       chatLog.ModelAgent.Path,
			Weight:     chatLog.ModelAgent.Weight,
			Remark:     chatLog.ModelAgent.Remark,
		}
	}

	if chatLog.FallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     chatLog.FallbackModelAgent.Id,
			ModelAgentName: chatLog.FallbackModelAgent.Name,
		}
	}

	if chatLog.FallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = chatLog.FallbackModel.Model
		chat.FallbackConfig.ModelName = chatLog.FallbackModel.Name
	}

	if chatLog.Key != nil {
		chat.Key = chatLog.Key.Key
	}

	if chatLog.CompletionsRes.Error != nil {

		chat.ErrMsg = chatLog.CompletionsRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(chatLog.CompletionsRes.Error, &openaiApiError) {
			chat.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(chatLog.CompletionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "messages") {
		for _, message := range chatLog.CompletionsReq.Messages {
			chat.Messages = append(chat.Messages, mcommon.Message{
				Role:    message.Role,
				Content: gconv.String(message.Content),
			})
		}
	}

	if chatLog.RetryInfo != nil {

		chat.IsRetry = chatLog.RetryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    chatLog.RetryInfo.IsRetry,
			RetryCount: chatLog.RetryInfo.RetryCount,
			ErrMsg:     chatLog.RetryInfo.ErrMsg,
		}

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = chatLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sRealtime SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			chatLog.CompletionsReq.Messages = []smodel.ChatCompletionMessage{{
				Role:    sconsts.ROLE_SYSTEM,
				Content: err.Error(),
			}}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sRealtime SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, chatLog, retry...)
	}
}
