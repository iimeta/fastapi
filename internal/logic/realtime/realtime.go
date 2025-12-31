package realtime

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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
		consts.TRACE_ID: {gtrace.GetTraceID(ctx)},
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

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ChatCompletionReq: smodel.ChatCompletionRequest{Stream: true},
					Error:             err,
					RetryInfo:         retryInfo,
					ConnTime:          connTime,
					Duration:          duration,
					TotalTime:         totalTime,
					InternalTime:      internalTime,
					EnterTime:         enterTime,
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

	response, err := common.NewRealtimeClient(ctx, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path).Realtime(ctx, requestChan)
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

					common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
						ChatCompletionReq: smodel.ChatCompletionRequest{Stream: true},
						Error:             response.Error,
						RetryInfo:         retryInfo,
						ConnTime:          connTime,
						Duration:          duration,
						TotalTime:         totalTime,
						InternalTime:      internalTime,
						EnterTime:         enterTime,
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

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
					internalTime := gtime.TimestampMilli() - enterTime - totalTime

					common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
						ChatCompletionReq: smodel.ChatCompletionRequest{Stream: true, Messages: []smodel.ChatCompletionMessage{{Content: message}}},
						Completion:        completion,
						Usage:             usage,
						Error:             err,
						RetryInfo:         retryInfo,
						ConnTime:          response.ConnTime,
						Duration:          response.Duration,
						TotalTime:         response.TotalTime,
						InternalTime:      internalTime,
						EnterTime:         enterTime,
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
