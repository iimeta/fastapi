package realtime

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
	sdk "github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
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
	"io"
	"math"
	"net/http"
	"slices"
	"time"
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
		client    *sdk.RealtimeClient
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

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &sdkm.ChatCompletionRequest{Stream: true}, completionsRes, retryInfo, false)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if client, err = common.NewRealtimeClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return err
	}

	requestChan := make(chan *sdkm.RealtimeRequest)

	response, err := client.Realtime(ctx, requestChan)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

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

					s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &sdkm.ChatCompletionRequest{Stream: true}, completionsRes, retryInfo, false)

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

				usage := &sdkm.Usage{
					PromptTokens:     realtimeResponse.Response.Usage.InputTokens,
					CompletionTokens: realtimeResponse.Response.Usage.OutputTokens,
					TotalTokens:      realtimeResponse.Response.Usage.TotalTokens,
				}

				message := responseMessage
				completion := responseCompletion
				totalTokens := 0

				typ := ""
				if len(realtimeResponse.Response.Output) > 0 {
					if len(realtimeResponse.Response.Output[0].Content) > 0 {
						typ = realtimeResponse.Response.Output[0].Content[0].Type
					} else {
						typ = realtimeResponse.Response.Output[0].Type
					}
				}

				if typ == "text" || typ == "function_call" {
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.RealtimeQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.RealtimeQuota.TextQuota.CompletionRatio))
				} else {
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.RealtimeQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.RealtimeQuota.AudioQuota.CompletionRatio))
				}

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key); err != nil {
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
					completionsRes.Usage.TotalTokens = totalTokens

					s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &sdkm.ChatCompletionRequest{Stream: true, Messages: []sdkm.ChatCompletionMessage{{Content: message}}}, completionsRes, retryInfo, false)

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

		requestChan <- &sdkm.RealtimeRequest{
			MessageType: messageType,
			Message:     message,
		}
	}
}

// 保存日志
func (s *sRealtime) SaveLog(ctx context.Context, reqModel, realModel *model.Model, modelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sRealtime SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && (errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) || errors.Is(completionsRes.Error, errors.ERR_MODEL_DISABLED)) {
		return
	}

	chat := do.Chat{
		TraceId:          gctx.CtxId(ctx),
		UserId:           service.Session().GetUserId(ctx),
		AppId:            service.Session().GetAppId(ctx),
		IsSmartMatch:     isSmartMatch,
		Stream:           completionsReq.Stream,
		PromptTokens:     completionsRes.Usage.PromptTokens,
		CompletionTokens: completionsRes.Usage.CompletionTokens,
		TotalTokens:      completionsRes.Usage.TotalTokens,
		ConnTime:         completionsRes.ConnTime,
		Duration:         completionsRes.Duration,
		TotalTime:        completionsRes.TotalTime,
		InternalTime:     completionsRes.InternalTime,
		ReqTime:          completionsRes.EnterTime,
		ReqDate:          gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:         g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:         g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:          util.GetLocalIp(),
		Status:           1,
		Host:             g.RequestFromCtx(ctx).GetHost(),
	}

	if config.Cfg.Log.Open && len(completionsReq.Messages) > 0 && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {
		chat.Prompt = gconv.String(completionsReq.Messages[len(completionsReq.Messages)-1].Content)
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") {
		chat.Completion = completionsRes.Completion
	}

	if reqModel != nil {
		chat.Corp = reqModel.Corp
		chat.ModelId = reqModel.Id
		chat.Name = reqModel.Name
		chat.Model = reqModel.Model
		chat.Type = reqModel.Type
		chat.RealtimeQuota = reqModel.RealtimeQuota
		if completionsRes.Type == "text" {
			chat.TextQuota = reqModel.RealtimeQuota.TextQuota
		} else {
			chat.TextQuota.BillingMethod = reqModel.RealtimeQuota.AudioQuota.BillingMethod
			chat.TextQuota.PromptRatio = reqModel.RealtimeQuota.AudioQuota.PromptRatio
			chat.TextQuota.CompletionRatio = reqModel.RealtimeQuota.AudioQuota.CompletionRatio
			chat.TextQuota.FixedQuota = reqModel.RealtimeQuota.AudioQuota.FixedQuota
		}
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
	}

	if chat.IsEnableModelAgent && modelAgent != nil {
		chat.ModelAgentId = modelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			Corp:    modelAgent.Corp,
			Name:    modelAgent.Name,
			BaseUrl: modelAgent.BaseUrl,
			Path:    modelAgent.Path,
			Weight:  modelAgent.Weight,
			Remark:  modelAgent.Remark,
			Status:  modelAgent.Status,
		}
	}

	if fallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = fallbackModel.Model
		chat.FallbackConfig.ModelName = fallbackModel.Name
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

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "messages") {
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
		logger.Errorf(ctx, "sRealtime SaveLog error: %v", err)

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sRealtime SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, modelAgent, fallbackModelAgent, fallbackModel, key, completionsReq, completionsRes, retryInfo, isSmartMatch, retry...)
	}
}
