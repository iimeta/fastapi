package bailian

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	sconsts "github.com/iimeta/fastapi-sdk/v2/consts"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
)

type sBailian struct{}

func init() {
	service.RegisterBailian(New())
}

func New() service.IBailian {
	return &sBailian{}
}

// ImageGenerations
func (s *sBailian) ImageGenerations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBailian ImageGenerations time: %d", gtime.TimestampMilli()-now)
	}()

	// 官方格式请求体转为标准请求, 仅用于计费与日志
	params, err := common.NewConverter(ctx, sconsts.PROVIDER_BAILIAN).ConvImageGenerationsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sBailian ImageGenerations ConvImageGenerationsRequest error: %v", err)
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

	j := gjson.New(data)

	if mak.RealModel != nil && !gstr.Contains(mak.RealModel.Model, "*") {
		_ = j.Set("model", mak.RealModel.Model)
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		reqModel := j.Get("model").String()
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == reqModel {
				logger.Infof(ctx, "sBailian ImageGenerations request.Model: %s replaced %s", reqModel, mak.ModelAgent.TargetModels[i])
				_ = j.Set("model", mak.ModelAgent.TargetModels[i])
				mak.RealModel.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	body := j.MustToJson()

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

	// 官方响应转为标准响应, 取出生成张数与实际分辨率用于计费
	if response, e := common.NewConverter(ctx, sconsts.PROVIDER_BAILIAN).ConvImageGenerationsResponse(ctx, responseBytes); e != nil {
		logger.Errorf(ctx, "sBailian ImageGenerations ConvImageGenerationsResponse error: %v", e)
	} else {
		imageResponse = response
	}

	imageResponse.TotalTime = totalTime
	imageResponse.ResponseBytes = responseBytes
	imageResponse.ResponseHeaders = responseHeader

	common.WritePassthroughHeaders(ctx, mak.Passthrough, responseHeader)

	return responseBytes, nil
}

// ImageGenerationsAsync
func (s *sBailian) ImageGenerationsAsync(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBailian ImageGenerationsAsync time: %d", gtime.TimestampMilli()-now)
	}()

	params, err := common.NewConverter(ctx, sconsts.PROVIDER_BAILIAN).ConvImageGenerationsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sBailian ImageGenerationsAsync ConvImageGenerationsRequest error: %v", err)
		return response, err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Endpoint:           consts.ENDPOINT_IMAGE_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo   *mcommon.Retry
		requestData map[string]any
	)

	imageId := "image_" + gtrace.GetTraceID(ctx)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := smodel.Usage{}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					Action:                 consts.ACTION_GENERATIONS,
					IsAsync:                true,
					ImageId:                imageId,
					RequestData:            requestData,
					Usage:                  &usage,
					Error:                  err,
					RetryInfo:              retryInfo,
					TotalTime:              response.TotalTime,
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
		return response, err
	}

	// 落库的请求数据即官方格式请求体, 后续由后台任务原样提交给上游, async 为本系统参数需剔除
	j := gjson.New(data)
	_ = j.Remove("async")

	if mak.RealModel != nil && !gstr.Contains(mak.RealModel.Model, "*") {
		_ = j.Set("model", mak.RealModel.Model)
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		reqModel := j.Get("model").String()
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == reqModel {
				logger.Infof(ctx, "sBailian ImageGenerationsAsync request.Model: %s replaced %s", reqModel, mak.ModelAgent.TargetModels[i])
				_ = j.Set("model", mak.ModelAgent.TargetModels[i])
				mak.RealModel.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	requestData = util.ConvToMap(j.MustToJson())

	response = smodel.ImageJobResponse{
		Id:           imageId,
		Object:       "image",
		Model:        mak.ReqModel.Name,
		Status:       "queued",
		Progress:     0,
		CreatedAt:    time.Now().Unix(),
		N:            params.N,
		Size:         params.Size,
		Prompt:       params.Prompt,
		OutputFormat: "png",
	}

	return response, nil
}
