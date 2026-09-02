package volcengine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
)

// ImageGenerations
func (s *sVolcEngine) ImageGenerations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine ImageGenerations time: %d", gtime.TimestampMilli()-now)
	}()

	var params smodel.ImageGenerationRequest
	if err = json.Unmarshal(data, &params); err != nil {
		logger.Errorf(ctx, "sVolcEngine ImageGenerations unmarshal request error: %v", err)
		return nil, err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Endpoint:           consts.ENDPOINT_IMAGE_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		imageResponse  smodel.ImageResponse
		retryInfo      *mcommon.Retry
		responseHeader http.Header
		totalTime      int64
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime
		usage := imageResponse.Usage

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					ImageResponse:          imageResponse,
					Action:                 consts.ACTION_GENERATIONS,
					Usage:                  &usage,
					Error:                  err,
					RetryInfo:              retryInfo,
					TotalTime:              totalTime,
					InternalTime:           internalTime,
					EnterTime:              enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	body := data
	j := gjson.New(body)

	if mak.RealModel != nil && !gstr.Contains(mak.RealModel.Model, "*") {
		_ = j.Set("model", mak.RealModel.Model)
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		reqModel := j.Get("model").String()
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == reqModel {
				logger.Infof(ctx, "sVolcEngine ImageGenerations request.Model: %s replaced %s", reqModel, mak.ModelAgent.TargetModels[i])
				_ = j.Set("model", mak.ModelAgent.TargetModels[i])
				mak.RealModel.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	if config.Cfg.ImageStorage.Open {
		_ = j.Set("response_format", "b64_json")
	}

	body = j.MustToJson()

	responseBytes, responseHeader, err = common.NewAdapterOfficial(ctx, mak, false).ImageGenerationsOfficial(ctx, body)
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
								return s.ImageGenerations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.ImageGenerations(g.RequestFromCtx(ctx).GetCtx(), data, nil, fallbackModel)
							}
						}
					}

					return nil, err
				}
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.ImageGenerations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return nil, err
	}

	totalTime = gtime.TimestampMilli() - now
	imageResponse.TotalTime = totalTime
	imageResponse.ResponseBytes = responseBytes
	imageResponse.ResponseHeaders = responseHeader

	if e := json.Unmarshal(responseBytes, &imageResponse); e != nil {
		logger.Errorf(ctx, "sVolcEngine ImageGenerations unmarshal response error: %v", e)
	}

	common.WritePassthroughHeaders(ctx, mak.Passthrough, responseHeader)

	return responseBytes, nil
}

// ImageGenerationsStream
func (s *sVolcEngine) ImageGenerationsStream(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine ImageGenerationsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var params smodel.ImageGenerationRequest
	if err = json.Unmarshal(data, &params); err != nil {
		logger.Errorf(ctx, "sVolcEngine ImageGenerationsStream unmarshal request error: %v", err)
		return err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Endpoint:           consts.ENDPOINT_IMAGE_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		imageResponse smodel.ImageResponse
		usage         *smodel.Usage
		connTime      int64
		duration      int64
		totalTime     int64
		retryInfo     *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime
		if usage == nil {
			usage = new(imageResponse.Usage)
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageResponse.TotalTime = totalTime

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					ImageResponse:          imageResponse,
					Action:                 consts.ACTION_GENERATIONS,
					Usage:                  usage,
					Error:                  err,
					RetryInfo:              retryInfo,
					ConnTime:               connTime,
					Duration:               duration,
					TotalTime:              totalTime,
					InternalTime:           internalTime,
					EnterTime:              enterTime,
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

	j := gjson.New(data)

	if mak.RealModel != nil && !gstr.Contains(mak.RealModel.Model, "*") {
		_ = j.Set("model", mak.RealModel.Model)
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		reqModel := j.Get("model").String()
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == reqModel {
				logger.Infof(ctx, "sVolcEngine ImageGenerationsStream request.Model: %s replaced %s", reqModel, mak.ModelAgent.TargetModels[i])
				_ = j.Set("model", mak.ModelAgent.TargetModels[i])
				mak.RealModel.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	_ = j.Set("stream", true)
	body := j.MustToJson()

	response, err := common.NewAdapterOfficial(ctx, mak, true).ImageGenerationsStreamOfficial(ctx, body)
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
								return s.ImageGenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.ImageGenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, nil, fallbackModel)
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

			return s.ImageGenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		item := <-response

		connTime = item.ConnTime
		duration = item.Duration
		totalTime = item.TotalTime

		if item.Error != nil {

			if errors.Is(item.Error, io.EOF) {
				return nil
			}

			err = item.Error
			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			if _, isDisabled := common.IsNeedRetry(err); isDisabled {
				if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
					service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			return err
		}

		if item.ResponseHeaders != nil {
			common.WritePassthroughHeaders(ctx, mak.Passthrough, item.ResponseHeaders)
		}

		if len(item.Data) > 0 {
			imageResponse.Data = append(imageResponse.Data, item.Data...)
		}

		if item.Usage.TotalTokens != 0 || item.Usage.InputTokens != 0 || item.Usage.OutputTokens != 0 {
			usage = new(item.Usage)
			imageResponse.Usage = item.Usage
		}

		if err = util.SSEServer(ctx, string(item.ResponseBytes), item.Event); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}
