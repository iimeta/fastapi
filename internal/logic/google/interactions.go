package google

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/v2/consts"
	sggoogle "github.com/iimeta/fastapi-sdk/v2/google"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func (s *sGoogle) Interactions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.GoogleInteractionRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGoogle Interactions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params          = convInteractionsToChatCompletionRequest(request)
		mak             = newInteractionsMAK(params, fallbackModelAgent, fallbackModel)
		retryInfo       *mcommon.Retry
		attempt         *common.AttemptLog
		imageFilePaths  []string
		imageExpiresAt  int64
		storedImageData []smodel.ImageResponseData
		chatRes         smodel.ChatCompletionResponse
	)

	defer func() {

		if mak.ReqModel != nil && mak.RealModel != nil {

			if mak.ReqModel.IsEnableForward {
				chatRes.Model = mak.ReqModel.Model
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				after := &mcommon.AfterHandler{
					ChatCompletionReq: params,
					ChatCompletionRes: chatRes,
					Action:            consts.ACTION_INTERACTIONS,
					Usage:             chatRes.Usage,
					Error:             err,
					RetryInfo:         retryInfo,
					ConnTime:          response.ConnTime,
					Duration:          response.Duration,
					TotalTime:         response.TotalTime,
				}

				if mak.ReqModel.Type == 8 {
					body := interactionRequestBody(ctx, request)
					after.Action = consts.ACTION_CREATE
					after.IsAsync = true
					after.VideoId = response.Id
					after.Prompt = extractInteractionPrompt(body)
					after.Seconds, after.Size, after.VideoMode = extractInteractionVideoMeta(body)
					after.RequestData = util.ConvToMap(body)
					after.ResponseData = util.ConvToMap(response.ResponseBytes)
				}

				if mak.ReqModel.Type == 2 {

					if after.ImageGenerationRequest, err = common.NewConverter(ctx, sconsts.PROVIDER_GOOGLE).ConvImageGenerationsRequest(ctx, request.GetBody()); err != nil {
						logger.Error(ctx, err)
					}

					if after.ImageGenerationRequest.Model == "" {
						after.ImageGenerationRequest.Model = mak.ReqModel.Model
					}

					if after.ImageResponse, err = common.NewConverter(ctx, sconsts.PROVIDER_GOOGLE).ConvImageGenerationsResponse(ctx, response.ResponseBytes); err != nil {
						logger.Error(ctx, err)
					}

					if len(storedImageData) > 0 {
						after.ImageResponse.Data = storedImageData
					} else if logData := extractGoogleInteractionImageDataForLog(response.ResponseBytes); len(logData) > 0 {
						after.ImageResponse.Data = normalizeImageDataUrls(logData)
					} else {
						after.ImageResponse.Data = normalizeImageDataUrls(after.ImageResponse.Data)
					}
					after.ImageFilePaths = imageFilePaths
					after.ImageExpiresAt = imageExpiresAt
				}

				common.FinishAttempt(ctx, now, attempt, after)
				common.AfterHandler(ctx, mak, after)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if mak.ReqModel.Type == 8 {
		return s.createQueuedInteraction(ctx, request, mak)
	}

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sGoogle Interactions params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]any)
				if err = json.Unmarshal(body, &data); err != nil {
					logger.Error(ctx, err)
					return response, err
				}

				if _, ok := data["model"]; ok {
					data["model"] = mak.RealModel.Model
				}

				body = gjson.MustEncode(data)

				break
			}
		}
	}

	if rewritten, rewriteErr := rewritePreviousInteractionId(ctx, body); rewriteErr != nil {
		logger.Error(ctx, rewriteErr)
		return response, rewriteErr
	} else if len(rewritten) > 0 {
		body = rewritten
		g.RequestFromCtx(ctx).SetCtxVar("google_interaction_body", body)
	}

	resetGenerateContentPath(mak)

	response, err = common.NewAdapterGoogle(ctx, mak, false).Interactions(ctx, body)
	if err != nil {
		logger.Error(ctx, err)

		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.AgentTotal, len(retry)) {

				if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
					service.Session().SaveModelAgentBillingMethod(ctx, 1)
					retry = []int{}
				} else {

					if mak.RealModel.IsEnableFallback {

						if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id && fallbackModelAgent == nil {
							if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}

								attempt = common.CaptureAttempt(ctx, now, response.TotalTime, response.ConnTime, response.Duration)
								return s.Interactions(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}

								attempt = common.CaptureAttempt(ctx, now, response.TotalTime, response.ConnTime, response.Duration)
								return s.Interactions(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
							}
						}
					}

					return response, err
				}
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			attempt = common.CaptureAttempt(ctx, now, response.TotalTime, response.ConnTime, response.Duration)
			return s.Interactions(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	if response.Error.Code == 0 && response.ResponseBytes != nil {
		if chatRes, err = sggoogle.ParseInteractionChatCompletion(response.ResponseBytes, mak.RealModel.Model, false); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
		if chatRes.Usage == nil {
			chatRes.Usage = sggoogle.ParseInteractionUsage(response.Usage)
		}
	}

	if mak.ReqModel != nil && mak.ReqModel.Type == 2 && response.ResponseBytes != nil && common.ShouldCheckImageSize(mak) {
		if imgReq, convErr := common.NewConverter(ctx, sconsts.PROVIDER_GOOGLE).ConvImageGenerationsRequest(ctx, request.GetBody()); convErr != nil {
			logger.Error(ctx, convErr)
		} else if expectedW, expectedH, ok := common.ExpectedGoogleImageSize(imgReq.Quality, imgReq.AspectRatio); ok {
			if err = common.CheckGeneratedImageSize(ctx, expectedW, expectedH, extractGoogleInteractionImageDataForLog(response.ResponseBytes)); err != nil {
				logger.Error(ctx, err)

				if shouldRetry, retryCount := common.ExcludeAgentForImageSizeRetry(ctx, mak); shouldRetry {
					retryInfo = &mcommon.Retry{
						IsRetry:    true,
						RetryCount: retryCount,
						ErrMsg:     err.Error(),
					}

					attempt = common.CaptureAttempt(ctx, now, response.TotalTime, response.ConnTime, response.Duration)
					return s.Interactions(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, retry...)
				}

				return response, err
			}
		}
	}

	if mak.ReqModel != nil && mak.ReqModel.Type == 2 && response.ResponseBytes != nil {
		response.ResponseBytes, imageFilePaths, imageExpiresAt, storedImageData = saveGoogleInteractionImageStorage(ctx, response.ResponseBytes, 0)
	}

	return response, nil
}

func (s *sGoogle) InteractionsStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGoogle InteractionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params          = convInteractionsToChatCompletionRequest(request)
		mak             = newInteractionsMAK(params, fallbackModelAgent, fallbackModel)
		completion      string
		msgId           string
		connTime        int64
		duration        int64
		totalTime       int64
		usage           *smodel.Usage
		retryInfo       *mcommon.Retry
		attempt         *common.AttemptLog
		imageFilePaths  []string
		imageExpiresAt  int64
		storedImageData []smodel.ImageResponseData
		responseHeaders map[string][]string
	)

	params.Stream = true

	defer func() {

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				after := &mcommon.AfterHandler{
					ChatCompletionReq: params,
					Completion:        completion,
					MsgId:             msgId,
					Action:            consts.ACTION_INTERACTIONS,
					Usage:             usage,
					Error:             err,
					RetryInfo:         retryInfo,
					ConnTime:          connTime,
					Duration:          duration,
					TotalTime:         totalTime,
				}

				if mak.ReqModel.Type == 8 {
					body := interactionRequestBody(ctx, request)
					after.Action = consts.ACTION_CREATE
					after.IsAsync = true
					after.VideoId = msgId
					after.Prompt = extractInteractionPrompt(body)
					after.Seconds, after.Size, after.VideoMode = extractInteractionVideoMeta(body)
					after.RequestData = util.ConvToMap(body)
					after.ResponseData = map[string]any{"id": msgId, "object": "interaction", "status": "in_progress"}
				}

				if mak.ReqModel.Type == 2 {

					if after.ImageGenerationRequest, err = common.NewConverter(ctx, sconsts.PROVIDER_GOOGLE).ConvImageGenerationsRequest(ctx, request.GetBody()); err != nil {
						logger.Error(ctx, err)
					}

					if after.ImageGenerationRequest.Model == "" {
						after.ImageGenerationRequest.Model = mak.ReqModel.Model
					}

					after.ImageResponse = smodel.ImageResponse{
						Data:      normalizeImageDataUrls(storedImageData),
						TotalTime: totalTime,
					}
					after.ImageFilePaths = imageFilePaths
					after.ImageExpiresAt = imageExpiresAt
				}

				common.FinishAttempt(ctx, now, attempt, after)
				common.AfterHandler(ctx, mak, after)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if mak.ReqModel.Type == 8 {
		response, createErr := s.createQueuedInteraction(ctx, request, mak)
		if createErr != nil {
			return createErr
		}
		msgId = response.Id
		passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
		common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)
		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
		return nil
	}

	body := request.GetBody()

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sGoogle InteractionsStream params.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])

				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model

				data := make(map[string]any)
				if err = json.Unmarshal(body, &data); err != nil {
					logger.Error(ctx, err)
					return err
				}

				if _, ok := data["model"]; ok {
					data["model"] = mak.RealModel.Model
				}

				body = gjson.MustEncode(data)

				break
			}
		}
	}

	if rewritten, rewriteErr := rewritePreviousInteractionId(ctx, body); rewriteErr != nil {
		logger.Error(ctx, rewriteErr)
		return rewriteErr
	} else if len(rewritten) > 0 {
		body = rewritten
		g.RequestFromCtx(ctx).SetCtxVar("google_interaction_body", body)
	}

	resetGenerateContentPath(mak)

	response, err := common.NewAdapterGoogle(ctx, mak, true).InteractionsStream(ctx, body)
	if err != nil {
		logger.Error(ctx, err)

		totalTime = gtime.TimestampMilli() - now

		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.AgentTotal, len(retry)) {

				if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
					service.Session().SaveModelAgentBillingMethod(ctx, 1)
					retry = []int{}
				} else {

					if mak.RealModel.IsEnableFallback {

						if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id && fallbackModelAgent == nil {
							if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}

								attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
								return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}

								attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
								return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
							}
						}
					}

					return err
				}
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
			return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		res := <-response
		if res == nil {
			return nil
		}

		connTime = res.ConnTime
		duration = res.Duration
		totalTime = res.TotalTime
		if res.ResponseHeaders != nil {
			responseHeaders = res.ResponseHeaders
		}

		if res.Err != nil {

			if errors.Is(res.Err, io.EOF) {
				return nil
			}

			err = res.Err

			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			isRetry, isDisabled := common.IsNeedRetry(err)

			if isDisabled {
				if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
					service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if isRetry {

				if common.IsMaxRetry(mak.AgentTotal, len(retry)) {

					if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
						service.Session().SaveModelAgentBillingMethod(ctx, 1)
						retry = []int{}
					} else {

						if mak.RealModel.IsEnableFallback {

							if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id && fallbackModelAgent == nil {
								if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
									retryInfo = &mcommon.Retry{
										IsRetry:    true,
										RetryCount: len(retry),
										ErrMsg:     err.Error(),
									}

									attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
									return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
								}
							}

							if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
								if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
									retryInfo = &mcommon.Retry{
										IsRetry:    true,
										RetryCount: len(retry),
										ErrMsg:     err.Error(),
									}

									attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
									return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
								}
							}
						}

						if err = writeInteractionStreamError(ctx, res, err); err != nil {
							logger.Error(ctx, err)
						}
						return err
					}
				}

				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}

				attempt = common.CaptureAttempt(ctx, now, totalTime, connTime, duration)
				return s.InteractionsStream(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
			}

			if err = writeInteractionStreamError(ctx, res, err); err != nil {
				logger.Error(ctx, err)
			}
			return err
		}

		if responseHeaders != nil {
			common.WritePassthroughHeaders(ctx, mak.Passthrough, responseHeaders)
		}

		if msgId == "" {
			msgId = res.Id
		}

		chatRes, convErr := convInteractionBytesToChat(res.ResponseBytes, mak.RealModel.Model)
		if convErr != nil {
			logger.Error(ctx, convErr)
		} else {
			if len(chatRes.Choices) > 0 && chatRes.Choices[0].Delta != nil {
				completion += gconv.String(chatRes.Choices[0].Delta.Content)
			}
			if chatRes.Usage != nil {
				if usage == nil {
					usage = chatRes.Usage
				} else {
					if chatRes.Usage.PromptTokens != 0 {
						usage.PromptTokens = chatRes.Usage.PromptTokens
					}
					if chatRes.Usage.CompletionTokens != 0 {
						usage.CompletionTokens = chatRes.Usage.CompletionTokens
					}
					if chatRes.Usage.TotalTokens != 0 {
						usage.TotalTokens = chatRes.Usage.TotalTokens
					} else {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					}
					if chatRes.Usage.PromptTokensDetails.CachedTokens != 0 {
						usage.PromptTokensDetails = chatRes.Usage.PromptTokensDetails
					}
					if chatRes.Usage.OutputTokensDetails.ReasoningTokens != 0 {
						usage.OutputTokensDetails = chatRes.Usage.OutputTokensDetails
					}
				}
			}
		}

		if mak.ReqModel != nil && mak.ReqModel.Type == 2 && res.ResponseBytes != nil {
			var paths []string
			var exp int64
			var imgData []smodel.ImageResponseData
			res.ResponseBytes, paths, exp, imgData = saveGoogleInteractionImageStorage(ctx, res.ResponseBytes, len(imageFilePaths))
			if len(imgData) > 0 {
				imageFilePaths = append(imageFilePaths, paths...)
				storedImageData = append(storedImageData, imgData...)
				if exp > 0 {
					imageExpiresAt = exp
				}
			} else {
				for _, item := range extractGoogleInteractionImageDataForLog(res.ResponseBytes) {
					if item.Url != "" {
						storedImageData = append(storedImageData, item)
					}
				}
			}
		}

		eventName := res.SSEEvent
		if eventName == "" {
			eventName = res.EventType
		}

		if err = util.SSEServer(ctx, string(res.ResponseBytes), eventName); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

func (s *sGoogle) InteractionsRetrieve(ctx context.Context, id string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.GoogleInteractionRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGoogle InteractionsRetrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{Endpoint: consts.ENDPOINT_INTERACTIONS, FallbackModelAgent: fallbackModelAgent, FallbackModel: fallbackModel}
		retryInfo *mcommon.Retry
		attempt   *common.AttemptLog
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				after := &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					IsAsync:      true,
					VideoId:      id,
					RequestData:  map[string]any{"id": id},
					ResponseData: util.ConvToMap(response.ResponseBytes),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
				}
				common.FinishAttempt(ctx, now, attempt, after)
				common.AfterHandler(ctx, mak, after)
			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": id, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Interaction with id '"+id+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskVideo.Model
	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	status := taskVideo.Status
	switch status {
	case "queued":
		status = "in_progress"
	}

	payload := map[string]any{
		"object": "interaction",
		"id":     taskVideo.VideoId,
		"status": status,
		"model":  taskVideo.Model,
		"steps":  []any{},
	}

	if taskVideo.Status == "completed" && taskVideo.VideoUrl != "" {
		payload["steps"] = []any{
			map[string]any{
				"type": "model_output",
				"content": []any{
					map[string]any{
						"type":      "video",
						"mime_type": "video/mp4",
						"uri":       common.ResolveVideoUrl(taskVideo.VideoUrl),
					},
				},
			},
		}
	}

	if taskVideo.Status == "failed" && taskVideo.Error != nil {
		code, message := errors.ShieldError(ctx, taskVideo.Error.Code, taskVideo.Error.Message)
		payload["error"] = map[string]any{
			"code":    code,
			"message": message,
		}
	}

	response.Id = taskVideo.VideoId
	response.Object = "interaction"
	response.Status = status
	response.Model = taskVideo.Model
	response.ResponseBytes = gjson.MustEncode(payload)

	return response, nil
}

func (s *sGoogle) createQueuedInteraction(ctx context.Context, request *ghttp.Request, mak *common.MAK) (response smodel.GoogleInteractionRes, err error) {

	body := request.GetBody()
	if rewritten, rewriteErr := rewritePreviousInteractionId(ctx, body); rewriteErr != nil {
		return response, rewriteErr
	} else if len(rewritten) > 0 {
		body = rewritten
		g.RequestFromCtx(ctx).SetCtxVar("google_interaction_body", body)
	}

	videoId := "video_" + gtrace.GetTraceID(ctx)
	modelName := mak.ReqModel.Name
	if modelName == "" {
		modelName = mak.ReqModel.Model
	}

	payload := map[string]any{
		"id":     videoId,
		"object": "interaction",
		"status": "in_progress",
		"model":  modelName,
		"steps":  []any{},
	}

	response.Id = videoId
	response.Object = "interaction"
	response.Status = "in_progress"
	response.Model = modelName
	response.ResponseBytes = gjson.MustEncode(payload)

	return response, nil
}

func newInteractionsMAK(params smodel.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model) *common.MAK {
	return &common.MAK{
		Model:              params.Model,
		Endpoint:           consts.ENDPOINT_INTERACTIONS,
		Messages:           params.Messages,
		FallbackModelAgent: fallbackModelAgent,
		FallbackModel:      fallbackModel,
	}
}

func convInteractionsToChatCompletionRequest(request *ghttp.Request) smodel.ChatCompletionRequest {

	req := smodel.GoogleInteractionReq{}
	if err := gjson.Unmarshal(request.GetBody(), &req); err != nil {
		logger.Error(request.GetCtx(), err)
		return smodel.ChatCompletionRequest{}
	}

	messages := make([]smodel.ChatCompletionMessage, 0)

	if req.SystemInstruction != nil {
		if text := gconv.String(req.SystemInstruction); text != "" && text != "<nil>" {
			messages = append(messages, smodel.ChatCompletionMessage{
				Role:    sconsts.ROLE_SYSTEM,
				Content: text,
			})
		}
	}

	contents := make([]any, 0)
	switch input := req.Input.(type) {
	case string:
		if input != "" {
			contents = append(contents, g.MapStrAny{"type": "text", "text": input})
		}
	case []any:
		for _, item := range input {
			content, ok := item.(map[string]any)
			if !ok {
				contents = append(contents, g.MapStrAny{"type": "text", "text": gconv.String(item)})
				continue
			}
			switch gconv.String(content["type"]) {
			case "image":
				url := firstNonEmptyString(gconv.String(content["uri"]), gconv.String(content["url"]), gconv.String(content["data"]))
				contents = append(contents, g.MapStrAny{
					"type": "image_url",
					"image_url": g.MapStrAny{
						"url": url,
					},
				})
			case "video":
				url := firstNonEmptyString(gconv.String(content["uri"]), gconv.String(content["url"]), gconv.String(content["data"]))
				contents = append(contents, g.MapStrAny{
					"type": "video_url",
					"video_url": g.MapStrAny{
						"url": url,
					},
				})
			default:
				if text := gconv.String(content["text"]); text != "" {
					contents = append(contents, g.MapStrAny{"type": "text", "text": text})
				}
			}
		}
	default:
		if req.Input != nil {
			contents = append(contents, g.MapStrAny{"type": "text", "text": gconv.String(req.Input)})
		}
	}

	if len(contents) > 0 {
		messages = append(messages, smodel.ChatCompletionMessage{
			Role:    "user",
			Content: contents,
		})
	}

	chatCompletionRequest := smodel.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if chatCompletionRequest.Model == "" {
		chatCompletionRequest.Model = request.Get("model").String()
	}

	if config, ok := req.GenerationConfig.(map[string]any); ok {
		chatCompletionRequest.MaxTokens = gconv.Int(config["max_output_tokens"])
		chatCompletionRequest.Temperature = gconv.Float32(config["temperature"])
		chatCompletionRequest.TopP = gconv.Float32(config["top_p"])
		if thinkingLevel := gconv.String(config["thinking_level"]); thinkingLevel != "" {
			chatCompletionRequest.ReasoningEffort = thinkingLevel
			enableThinking := thinkingLevel != "minimal" && thinkingLevel != "none"
			chatCompletionRequest.EnableThinking = &enableThinking
		}
		if budget := config["thinking_budget"]; budget != nil {
			enableThinking := gconv.Int(budget) != 0
			chatCompletionRequest.EnableThinking = &enableThinking
		}
	}

	return chatCompletionRequest
}

func rewritePreviousInteractionId(ctx context.Context, body []byte) ([]byte, error) {

	data := make(map[string]any)
	if err := json.Unmarshal(body, &data); err != nil {
		return body, nil
	}

	previousId := gconv.String(data["previous_interaction_id"])
	if previousId == "" || !gstr.HasPrefix(previousId, "video_") {
		return body, nil
	}

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": previousId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.NewError(404, "invalid_request_error", "Interaction with id '"+previousId+"' not found.", "invalid_request_error", nil)
		}
		return nil, err
	}

	if taskVideo.JobId == "" || taskVideo.JobId == "-" || gstr.HasPrefix(taskVideo.JobId, "video_") {
		return nil, errors.NewError(400, "invalid_request_error", "previous interaction is not ready", "invalid_request_error", "previous_interaction_id")
	}

	data["previous_interaction_id"] = taskVideo.JobId
	return gjson.MustEncode(data), nil
}

func resetGenerateContentPath(mak *common.MAK) {
	if mak == nil {
		return
	}
	if strings.Contains(mak.Path, ":") || strings.Contains(mak.Path, "/models/") {
		mak.Path = ""
	}
}

func extractInteractionPrompt(body []byte) string {

	req := smodel.GoogleInteractionReq{}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}

	switch input := req.Input.(type) {
	case string:
		return input
	case []any:
		var prompt string
		for _, item := range input {
			if content, ok := item.(map[string]any); ok {
				if text := gconv.String(content["text"]); text != "" {
					prompt += text
				}
			}
		}
		return prompt
	default:
		return gconv.String(req.Input)
	}
}

func extractInteractionVideoMeta(body []byte) (seconds int, size, mode string) {

	req := smodel.GoogleInteractionReq{}
	if err := json.Unmarshal(body, &req); err != nil {
		return 0, "", "no_video_input"
	}

	hasVideoInput := false
	switch input := req.Input.(type) {
	case []any:
		for _, item := range input {
			if content, ok := item.(map[string]any); ok {
				typ := gconv.String(content["type"])
				if typ == "image" || typ == "video" {
					hasVideoInput = true
				}
			}
		}
	}

	if hasVideoInput {
		mode = "has_video_input"
	} else {
		mode = "no_video_input"
	}

	if format, ok := req.ResponseFormat.(map[string]any); ok {
		aspectRatio := gconv.String(format["aspect_ratio"])
		resolution := gconv.String(format["resolution"])
		if resolution != "" && aspectRatio != "" {
			size = consts.VIDEO_RESOLUTION_RATIO[resolution+aspectRatio]
		}
		if size == "" && aspectRatio != "" {
			size = aspectRatio
		}
		seconds = gconv.Int(format["duration"])
		if seconds == 0 {
			seconds = gconv.Int(format["seconds"])
		}
	}

	return seconds, size, mode
}

func convInteractionBytesToChat(data []byte, modelName string) (smodel.ChatCompletionResponse, error) {

	if len(data) == 0 {
		return smodel.ChatCompletionResponse{}, nil
	}

	response, err := sggoogle.ParseInteractionChatCompletion(data, modelName, true)
	if err != nil {
		return smodel.ChatCompletionResponse{}, err
	}

	if response.Model == "" {
		response.Model = modelName
	}

	return response, nil
}

func writeInteractionStreamError(ctx context.Context, res *smodel.GoogleInteractionRes, streamErr error) error {

	if res == nil {
		return streamErr
	}

	event := res.SSEEvent
	if event == "" {
		event = res.EventType
	}
	if event == "" {
		event = "error"
	}

	if len(res.ResponseBytes) > 0 {
		return util.SSEServer(ctx, string(res.ResponseBytes), event)
	}

	apiErr := errors.Error(ctx, streamErr)
	return util.SSEServer(ctx, gjson.MustEncodeString(apiErr), event)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func interactionRequestBody(ctx context.Context, request *ghttp.Request) []byte {
	if v := g.RequestFromCtx(ctx).GetCtxVar("google_interaction_body"); v != nil {
		if body, ok := v.Val().([]byte); ok && len(body) > 0 {
			return body
		}
	}
	return request.GetBody()
}
