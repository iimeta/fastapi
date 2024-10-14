package audio

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
func (s *sRealtime) Realtime(ctx context.Context, r *ghttp.Request, params model.RealtimeRequest, fallbackModel *model.Model, retry ...int) (err error) {

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
		client     *sdk.RealtimeClient
		reqModel   *model.Model
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
		baseUrl    string
		path       string
		agentTotal int
		keyTotal   int
		connTime   int64
		duration   int64
		totalTime  int64
		retryInfo  *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				realModel.ModelAgent = modelAgent

				completionsRes := &model.CompletionsRes{
					Error:        err,
					ConnTime:     connTime,
					Duration:     duration,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &sdkm.ChatCompletionRequest{Stream: true}, completionsRes, retryInfo, false)

			}); err != nil {
				logger.Error(ctx, err)
			}
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

	client, err = common.NewRealtimeClient(ctx, realModel, k.Key, baseUrl, path)
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

	requestChan := make(chan *sdkm.RealtimeRequest)

	response, err := client.Realtime(ctx, requestChan)
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
						return s.Realtime(ctx, r, params, fallbackModel)
					}
				}
				return err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Realtime(ctx, r, params, fallbackModel, append(retry, 1)...)
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
				service.Common().RecordError(ctx, realModel, k, modelAgent)

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					realModel.ModelAgent = modelAgent
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

					s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &sdkm.ChatCompletionRequest{Stream: true}, completionsRes, retryInfo, false)

				}); err != nil {
					logger.Error(ctx, err)
				}

				if err := conn.Close(); err != nil {
					logger.Error(ctx, err)
				}

				return
			}

			logger.Debugf(ctx, "Response messageType: %d, message: %s", response.MessageType, response.Message)

			realtimeResponse := new(model.RealtimeResponse)
			if err = gjson.Unmarshal(response.Message, &realtimeResponse); err != nil {
				logger.Errorf(ctx, "response.Message: %s, error: %v", response.Message, err)
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
				if realtimeResponse.Item.Content[0].Text != "" {
					responseCompletion = realtimeResponse.Item.Content[0].Text
				}
				if realtimeResponse.Item.Content[0].Transcript != "" {
					responseCompletion = realtimeResponse.Item.Content[0].Transcript
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

				if realtimeResponse.Response.Output[0].Content[0].Type == "text" {
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.RealtimeQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*reqModel.RealtimeQuota.TextQuota.CompletionRatio))
				} else {
					totalTokens = int(math.Ceil(float64(usage.PromptTokens)*reqModel.RealtimeQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*reqModel.RealtimeQuota.AudioQuota.CompletionRatio))
				}

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					realModel.ModelAgent = modelAgent
					enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
					internalTime := gtime.TimestampMilli() - enterTime - totalTime

					completionsRes := &model.CompletionsRes{
						Type:         realtimeResponse.Response.Output[0].Content[0].Type,
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

					s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &sdkm.ChatCompletionRequest{Stream: true, Messages: []sdkm.ChatCompletionMessage{{Content: message}}}, completionsRes, retryInfo, false)

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

		logger.Debugf(ctx, "Request messageType: %d, message: %s", messageType, message)

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
func (s *sRealtime) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int) {

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
		Host:         g.RequestFromCtx(ctx).GetHost(),
	}

	if len(completionsReq.Messages) > 0 && slices.Contains(config.Cfg.RecordLogs, "prompt") {
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
