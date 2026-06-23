package image

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	sconsts "github.com/iimeta/fastapi-sdk/v2/consts"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	sutil "github.com/iimeta/fastapi-sdk/v2/util"
	v1 "github.com/iimeta/fastapi/v2/api/image/v1"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/db"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type sImage struct{}

func init() {
	service.RegisterImage(New())
}

func New() service.IImage {
	return &sImage{}
}

// Generations
func (s *sImage) Generations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Generations time: %d", gtime.TimestampMilli()-now)
	}()

	params, err := common.NewConverter(ctx, sconsts.PROVIDER_OPENAI).ConvImageGenerationsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sImage Generations ConvImageGenerationsRequest error: %v", err)
		return response, err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo      *mcommon.Retry
		imageFilePaths []string
		imageExpiresAt int64
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					ImageResponse:          response,
					Action:                 consts.ACTION_GENERATIONS,
					ImageFilePaths:         imageFilePaths,
					ImageExpiresAt:         imageExpiresAt,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageGenerationRequest: params,
		}

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "image_generation")
		if spend.ImageGeneration.Pricing.Width > 0 && spend.ImageGeneration.Pricing.Height > 0 {
			if spend.ImageGeneration.Pricing.Quality != "" {
				params.Quality = spend.ImageGeneration.Pricing.Quality
			}
			params.Size = fmt.Sprintf("%dx%d", spend.ImageGeneration.Pricing.Width, spend.ImageGeneration.Pricing.Height)
		}
	}

	request := params

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sImage Generations request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	if config.Cfg.ImageStorage.Open {
		request.ResponseFormat = "b64_json"
	}

	response, err = common.NewAdapter(ctx, mak, false).ImageGenerations(ctx, gjson.MustEncode(request))
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
								return s.Generations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.Generations(g.RequestFromCtx(ctx).GetCtx(), data, nil, fallbackModel)
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

			return s.Generations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	imageFilePaths, imageExpiresAt = saveImageStorage(ctx, &response, params.OutputFormat)

	return response, nil
}

// GenerationsStream
func (s *sImage) GenerationsStream(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage GenerationsStream time: %d", gtime.TimestampMilli()-now)
	}()

	params, err := common.NewConverter(ctx, sconsts.PROVIDER_OPENAI).ConvImageGenerationsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sImage GenerationsStream ConvImageGenerationsRequest error: %v", err)
		return err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		imageResponse  smodel.ImageResponse
		usage          *smodel.Usage
		connTime       int64
		duration       int64
		totalTime      int64
		retryInfo      *mcommon.Retry
		imageFilePaths []string
		imageExpiresAt int64
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageResponse.TotalTime = totalTime

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					ImageResponse:          imageResponse,
					Action:                 consts.ACTION_GENERATIONS,
					ImageFilePaths:         imageFilePaths,
					ImageExpiresAt:         imageExpiresAt,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageGenerationRequest: params,
		}

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "image_generation")
		if spend.ImageGeneration.Pricing.Width > 0 && spend.ImageGeneration.Pricing.Height > 0 {
			if spend.ImageGeneration.Pricing.Quality != "" {
				params.Quality = spend.ImageGeneration.Pricing.Quality
			}
			params.Size = fmt.Sprintf("%dx%d", spend.ImageGeneration.Pricing.Width, spend.ImageGeneration.Pricing.Height)
		}
	}

	request := params

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sImage GenerationsStream request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	response, err := common.NewAdapter(ctx, mak, true).ImageGenerationsStream(ctx, gjson.MustEncode(request))
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
								return s.GenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.GenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, nil, fallbackModel)
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

			return s.GenerationsStream(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Error != nil {

			if errors.Is(response.Error, io.EOF) {
				imageFilePaths, imageExpiresAt = saveImageStorage(ctx, &imageResponse, params.OutputFormat)
				return nil
			}

			err = response.Error

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			if _, isDisabled := common.IsNeedRetry(err); isDisabled {
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

			return err
		}

		// 响应头透传
		if response.ResponseHeaders != nil {
			common.WritePassthroughHeaders(ctx, mak.Passthrough, response.ResponseHeaders)
		}

		if len(response.Data) > 0 {
			imageResponse.Data = response.Data
		}

		if response.Usage.TotalTokens != 0 || response.Usage.InputTokens != 0 || response.Usage.OutputTokens != 0 {
			usage = &response.Usage
		}

		if err = util.SSEServer(ctx, string(response.ResponseBytes), response.Event); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

// Edits
func (s *sImage) Edits(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Edits time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo      *mcommon.Retry
		imageFilePaths []string
		imageExpiresAt int64
	)

	if gstr.Contains(g.RequestFromCtx(ctx).Header.Get("Content-Type"), "multipart/form-data") {
		// form 文件方式提交, image 必填
		if params.Image == nil {
			return response, errors.ERR_MISSING_REQUIRED_PARAMETER_IMAGE
		}
	} else {
		// json 方式提交, images 必填
		if len(params.Images) == 0 {
			return response, errors.ERR_MISSING_REQUIRED_PARAMETER_IMAGES
		}
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageReq := smodel.ImageGenerationRequest{
					Prompt:         params.Prompt,
					Background:     params.Background,
					Model:          params.Model,
					N:              params.N,
					Quality:        params.Quality,
					ResponseFormat: params.ResponseFormat,
					Size:           params.Size,
					User:           params.User,
					Stream:         params.Stream,
				}

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: imageReq,
					ImageResponse:          response,
					Action:                 consts.ACTION_EDITS,
					ImageFilePaths:         imageFilePaths,
					ImageExpiresAt:         imageExpiresAt,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageEditRequest: params,
		}

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "image_generation")
		if spend.ImageGeneration.Pricing.Width > 0 && spend.ImageGeneration.Pricing.Height > 0 {
			if spend.ImageGeneration.Pricing.Quality != "" {
				params.Quality = spend.ImageGeneration.Pricing.Quality
			}
			params.Size = fmt.Sprintf("%dx%d", spend.ImageGeneration.Pricing.Width, spend.ImageGeneration.Pricing.Height)
		}
	}

	if !gstr.Contains(mak.RealModel.Model, "*") {
		params.Model = mak.RealModel.Model
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sImage Edits request.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])
				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model
				break
			}
		}
	}

	if config.Cfg.ImageStorage.Open {
		params.ResponseFormat = "b64_json"
	}

	response, err = common.NewAdapter(ctx, mak, false).ImageEdits(ctx, params)
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
								return s.Edits(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.Edits(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Edits(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	imageFilePaths, imageExpiresAt = saveImageStorage(ctx, &response, params.OutputFormat)

	return response, nil
}

// EditsStream
func (s *sImage) EditsStream(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage EditsStream time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		imageResponse  smodel.ImageResponse
		usage          *smodel.Usage
		connTime       int64
		duration       int64
		totalTime      int64
		retryInfo      *mcommon.Retry
		imageFilePaths []string
		imageExpiresAt int64
	)

	if gstr.Contains(g.RequestFromCtx(ctx).Header.Get("Content-Type"), "multipart/form-data") {
		// form 文件方式提交, image 必填
		if params.Image == nil {
			return errors.ERR_MISSING_REQUIRED_PARAMETER_IMAGE
		}
	} else {
		// json 方式提交, images 必填
		if len(params.Images) == 0 {
			return errors.ERR_MISSING_REQUIRED_PARAMETER_IMAGES
		}
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageResponse.TotalTime = totalTime

				imageReq := smodel.ImageGenerationRequest{
					Prompt:         params.Prompt,
					Background:     params.Background,
					Model:          params.Model,
					N:              params.N,
					Quality:        params.Quality,
					ResponseFormat: params.ResponseFormat,
					Size:           params.Size,
					User:           params.User,
					Stream:         params.Stream,
				}

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: imageReq,
					ImageResponse:          imageResponse,
					Action:                 consts.ACTION_EDITS,
					ImageFilePaths:         imageFilePaths,
					ImageExpiresAt:         imageExpiresAt,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageEditRequest: params,
		}

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "image_generation")
		if spend.ImageGeneration.Pricing.Width > 0 && spend.ImageGeneration.Pricing.Height > 0 {
			if spend.ImageGeneration.Pricing.Quality != "" {
				params.Quality = spend.ImageGeneration.Pricing.Quality
			}
			params.Size = fmt.Sprintf("%dx%d", spend.ImageGeneration.Pricing.Width, spend.ImageGeneration.Pricing.Height)
		}
	}

	if !gstr.Contains(mak.RealModel.Model, "*") {
		params.Model = mak.RealModel.Model
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == params.Model {
				logger.Infof(ctx, "sImage EditsStream request.Model: %s replaced %s", params.Model, mak.ModelAgent.TargetModels[i])
				params.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = params.Model
				break
			}
		}
	}

	response, err := common.NewAdapter(ctx, mak, true).ImageEditsStream(ctx, params)
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
								return s.EditsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.EditsStream(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.EditsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Error != nil {

			if errors.Is(response.Error, io.EOF) {
				imageFilePaths, imageExpiresAt = saveImageStorage(ctx, &imageResponse, params.OutputFormat)
				return nil
			}

			err = response.Error

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			if _, isDisabled := common.IsNeedRetry(err); isDisabled {
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

			return err
		}

		// 响应头透传
		if response.ResponseHeaders != nil {
			common.WritePassthroughHeaders(ctx, mak.Passthrough, response.ResponseHeaders)
		}

		if len(response.Data) > 0 {
			imageResponse.Data = response.Data
		}

		if response.Usage.TotalTokens != 0 || response.Usage.InputTokens != 0 || response.Usage.OutputTokens != 0 {
			usage = &response.Usage
		}

		if err = util.SSEServer(ctx, string(response.ResponseBytes), response.Event); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}

// GenerationsAsync
func (s *sImage) GenerationsAsync(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage GenerationsAsync time: %d", gtime.TimestampMilli()-now)
	}()

	params, err := common.NewConverter(ctx, sconsts.PROVIDER_OPENAI).ConvImageGenerationsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sImage GenerationsAsync ConvImageGenerationsRequest error: %v", err)
		return response, err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
	)

	imageId := "image_" + gtrace.GetTraceID(ctx)

	action := consts.ACTION_GENERATIONS
	if params.Image != nil || len(params.Images) > 0 {
		action = consts.ACTION_EDITS
		if err = checkAsyncEditImage(smodel.ImageEditRequest{Image: params.Image, Images: params.Images}); err != nil {
			logger.Errorf(ctx, "sImage GenerationsAsync checkAsyncEditImage error: %v", err)
			return response, err
		}
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := smodel.Usage{}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: params,
					Action:                 action,
					IsAsync:                true,
					ImageId:                imageId,
					RequestData:            util.ConvToMap(gjson.MustEncode(params)),
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

	response = smodel.ImageJobResponse{
		Id:           imageId,
		Object:       "image",
		Model:        mak.ReqModel.Name,
		Status:       "queued",
		Progress:     0,
		CreatedAt:    time.Now().Unix(),
		N:            params.N,
		Quality:      params.Quality,
		Size:         params.Size,
		Prompt:       params.Prompt,
		OutputFormat: params.OutputFormat,
	}

	return response, nil
}

// EditsAsync
func (s *sImage) EditsAsync(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage EditsAsync time: %d", gtime.TimestampMilli()-now)
	}()

	// 异步编辑仅支持图像URL或file_id, 不支持上传文件和base64
	if err = checkAsyncEditImage(params); err != nil {
		logger.Errorf(ctx, "sImage EditsAsync checkAsyncEditImage error: %v", err)
		return response, err
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
	)

	imageId := "image_" + gtrace.GetTraceID(ctx)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := smodel.Usage{}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageReq := smodel.ImageGenerationRequest{
					Prompt:         params.Prompt,
					Background:     params.Background,
					Model:          params.Model,
					N:              params.N,
					Quality:        params.Quality,
					ResponseFormat: params.ResponseFormat,
					Size:           params.Size,
					User:           params.User,
				}

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					ImageGenerationRequest: imageReq,
					Action:                 consts.ACTION_EDITS,
					IsAsync:                true,
					ImageId:                imageId,
					RequestData:            util.ConvToMap(gjson.MustEncode(params)),
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

	response = smodel.ImageJobResponse{
		Id:           imageId,
		Object:       "image",
		Model:        mak.ReqModel.Name,
		Status:       "queued",
		Progress:     0,
		CreatedAt:    time.Now().Unix(),
		N:            params.N,
		Quality:      params.Quality,
		Size:         params.Size,
		Prompt:       params.Prompt,
		OutputFormat: params.OutputFormat,
	}

	return response, nil
}

// 校验异步编辑的图像输入, 仅支持图像URL或file_id
func checkAsyncEditImage(params smodel.ImageEditRequest) error {

	isInvalid := func(s string) bool {
		return s == "" || gstr.HasPrefix(s, "data:")
	}

	if len(params.Images) > 0 {
		for _, image := range params.Images {
			if image.FileId != "" {
				continue
			}
			if isInvalid(image.ImageUrl) {
				return errors.NewError(400, "invalid_request_error", "Async edits only support image url or file_id.", "invalid_request_error", "image")
			}
		}
	} else {

		switch v := params.Image.(type) {
		case string:
			if isInvalid(v) {
				return errors.NewError(400, "invalid_request_error", "Async edits only support image url or file_id.", "invalid_request_error", "image")
			}
		case []any:
			for _, item := range v {
				s, ok := item.(string)
				if !ok || isInvalid(s) {
					return errors.NewError(400, "invalid_request_error", "Async edits only support image url or file_id.", "invalid_request_error", "image")
				}
			}
		default:
			return errors.NewError(400, "invalid_request_error", "Async edits only support image url or file_id.", "invalid_request_error", "image")
		}
	}

	// 校验mask: 异步仅支持URL或file_id, 不支持上传文件和base64
	if params.Mask != nil {
		switch v := params.Mask.(type) {
		case string:
			// 字符串视为URL
			if isInvalid(v) {
				return errors.NewError(400, "invalid_request_error", "Async edits only support mask url or file_id.", "invalid_request_error", "mask")
			}
		case smodel.ImageEditImage:
			if v.FileId == "" && isInvalid(v.ImageUrl) {
				return errors.NewError(400, "invalid_request_error", "Async edits only support mask url or file_id.", "invalid_request_error", "mask")
			}
		case map[string]any:
			fileId, _ := v["file_id"].(string)
			imageUrl, _ := v["image_url"].(string)
			if fileId == "" && isInvalid(imageUrl) {
				return errors.NewError(400, "invalid_request_error", "Async edits only support mask url or file_id.", "invalid_request_error", "mask")
			}
		default:
			return errors.NewError(400, "invalid_request_error", "Async edits only support mask url or file_id.", "invalid_request_error", "mask")
		}
	}

	return nil
}

// List
func (s *sImage) List(ctx context.Context, params *v1.ListReq) (response smodel.ImageListResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage List time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_LIST,
					RequestData:  util.ConvToMap(params.ImageListRequest),
					ResponseData: util.ConvToMap(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	limit := params.Limit

	if limit > 1000 {
		err = errors.NewError(404, "integer_above_max_value", fmt.Sprintf("Invalid 'limit': integer above maximum value. Expected a value <= 1000, but got %d instead.", params.Limit), "invalid_request_error", "limit")
		return response, err
	} else if limit == 0 {
		limit = 1000
	}

	filter := bson.M{
		"creator":    service.Session().GetSecretKey(ctx),
		"status":     bson.M{"$nin": []string{"deleted", "expired"}},
		"created_at": bson.M{"$gt": time.Now().Add(-24 * time.Hour).UnixMilli()},
	}

	if params.After != "" {

		taskImage, err := dao.TaskImage.FindOne(ctx, bson.M{"image_id": params.After, "creator": service.Session().GetSecretKey(ctx)})
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = errors.NewError(404, "invalid_request_error", "Image with id '"+params.After+"' not found.", "invalid_request_error", nil)
			}
			logger.Error(ctx, err)
			return response, err
		}

		filter["created_at"] = bson.M{"$lte": taskImage.CreatedAt}

		if params.Order == "asc" {
			filter["created_at"] = bson.M{"$gte": taskImage.CreatedAt}
		}

		filter["_id"] = bson.M{"$ne": taskImage.Id}
	}

	sort := "-created_at"
	if params.Order == "asc" {
		sort = "created_at"
	}

	paging := &db.Paging{
		Page:     1,
		PageSize: limit,
	}

	results, err := dao.TaskImage.FindByPage(ctx, paging, filter, &dao.FindOptions{SortFields: []string{sort}})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if len(results) == 0 {
		response = smodel.ImageListResponse{
			Object: "list",
			Data:   make([]smodel.ImageJobResponse, 0),
		}
		return response, nil
	}

	mak.Model = results[0].Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.ImageListResponse{
		Object:  "list",
		FirstId: &results[0].ImageId,
		LastId:  &results[len(results)-1].ImageId,
		HasMore: paging.PageCount > 1,
	}

	for _, result := range results {

		imageJobResponse := smodel.ImageJobResponse{
			Id:        result.ImageId,
			Object:    "image",
			Model:     result.Model,
			Status:    result.Status,
			Progress:  result.Progress,
			CreatedAt: result.CreatedAt / 1000,
			Size:      result.Size,
			Quality:   result.Quality,
			N:         result.N,
			Prompt:    result.Prompt,
			Error:     result.Error,
		}

		if result.CompletedAt != 0 {
			imageJobResponse.CompletedAt = &result.CompletedAt
		}

		if result.ExpiresAt != 0 {
			imageJobResponse.ExpiresAt = &result.ExpiresAt
		}

		if config.Cfg.ImageTask.IsEnableStorage && result.ImageUrl != "" {

			if config.Cfg.ImageTask.StorageBaseUrl != "" {
				if gstr.HasSuffix(config.Cfg.ImageTask.StorageBaseUrl, "/") {
					result.ImageUrl = gstr.TrimLeftStr(result.ImageUrl, "/")
				} else if !gstr.HasPrefix(result.ImageUrl, "/") {
					result.ImageUrl = "/" + result.ImageUrl
				}
			}

			imageJobResponse.ImageUrl = config.Cfg.ImageTask.StorageBaseUrl + result.ImageUrl
		}

		if config.Cfg.ImageTask.IsEnableStorage && len(result.ImageUrls) > 0 {
			imageJobResponse.ImageUrls = make([]string, 0, len(result.ImageUrls))
			for _, u := range result.ImageUrls {
				imageJobResponse.ImageUrls = append(imageJobResponse.ImageUrls, buildStorageUrl(u))
			}
		}

		response.Data = append(response.Data, imageJobResponse)
	}

	return response, nil
}

// Retrieve
func (s *sImage) Retrieve(ctx context.Context, params smodel.ImageRetrieveRequest) (response smodel.ImageJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Retrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					ImageId:      params.ImageId,
					RequestData:  util.ConvToMap(params),
					ResponseData: util.ConvToMap(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskImages, err := dao.TaskImage.Find(ctx, bson.M{"image_id": params.ImageId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	taskImage := pickRepresentativeTaskImage(taskImages)
	if taskImage == nil {
		err = errors.NewError(404, "invalid_request_error", "Image with id '"+params.ImageId+"' not found.", "invalid_request_error", nil)
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskImage.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.ImageJobResponse{
		Id:        taskImage.ImageId,
		Object:    "image",
		Model:     taskImage.Model,
		Status:    taskImage.Status,
		Progress:  taskImage.Progress,
		CreatedAt: taskImage.CreatedAt / 1000,
		Size:      taskImage.Size,
		Quality:   taskImage.Quality,
		N:         taskImage.N,
		Prompt:    taskImage.Prompt,
		Error:     taskImage.Error,
	}

	if taskImage.CompletedAt != 0 {
		response.CompletedAt = &taskImage.CompletedAt
	}

	if taskImage.ExpiresAt != 0 {
		response.ExpiresAt = &taskImage.ExpiresAt
	}

	if config.Cfg.ImageTask.IsEnableStorage && taskImage.ImageUrl != "" {

		if config.Cfg.ImageTask.StorageBaseUrl != "" {
			if gstr.HasSuffix(config.Cfg.ImageTask.StorageBaseUrl, "/") {
				taskImage.ImageUrl = gstr.TrimLeftStr(taskImage.ImageUrl, "/")
			} else if !gstr.HasPrefix(taskImage.ImageUrl, "/") {
				taskImage.ImageUrl = "/" + taskImage.ImageUrl
			}
		}

		response.ImageUrl = config.Cfg.ImageTask.StorageBaseUrl + taskImage.ImageUrl
	}

	if config.Cfg.ImageTask.IsEnableStorage && len(taskImage.ImageUrls) > 0 {
		response.ImageUrls = make([]string, 0, len(taskImage.ImageUrls))
		for _, u := range taskImage.ImageUrls {
			response.ImageUrls = append(response.ImageUrls, buildStorageUrl(u))
		}
	}

	return response, nil
}

// Delete
func (s *sImage) Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.ImageJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Delete time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_DELETE,
					ImageId:      params.ImageId,
					RequestData:  util.ConvToMap(params.ImageDeleteRequest),
					ResponseData: util.ConvToMap(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskImages, err := dao.TaskImage.Find(ctx, bson.M{"image_id": params.ImageId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	taskImage := pickRepresentativeTaskImage(taskImages)
	if taskImage == nil {
		err = errors.NewError(404, "invalid_request_error", "Image with id '"+params.ImageId+"' not found.", "invalid_request_error", nil)
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskImage.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	// 同一 image_id 只要有进行中的任务, 整体拒绝删除
	for _, t := range taskImages {
		if t.Status == "in_progress" {
			err = errors.NewError(409, "invalid_request_error", "Image with id '"+params.ImageId+"' is in progress and cannot be deleted.", "invalid_request_error", nil)
			logger.Error(ctx, err)
			return response, err
		}
	}

	// 删除同一 image_id 的全部记录: 清理已落盘文件并置为 deleted
	for _, t := range taskImages {

		if t.Status == "deleted" {
			continue
		}

		if config.Cfg.ImageTask.IsEnableStorage && t.FilePath != "" {
			if err := gfile.RemoveFile(t.FilePath); err != nil {
				logger.Error(ctx, err)
			}
		}

		if config.Cfg.ImageTask.IsEnableStorage && len(t.FilePaths) > 0 {
			for _, fp := range t.FilePaths {
				if fp != "" && fp != t.FilePath {
					if err := gfile.RemoveFile(fp); err != nil {
						logger.Error(ctx, err)
					}
				}
			}
		}

		if err := dao.TaskImage.UpdateById(ctx, t.Id, bson.M{"status": "deleted", "image_url": "", "image_urls": nil, "file_name": "", "file_names": nil, "file_path": "", "file_paths": nil}); err != nil {
			logger.Error(ctx, err)
		}
	}

	response = smodel.ImageJobResponse{
		Id:        taskImage.ImageId,
		Object:    "image.deleted",
		Model:     taskImage.Model,
		Status:    "deleted",
		Progress:  taskImage.Progress,
		CreatedAt: taskImage.CreatedAt / 1000,
		Size:      taskImage.Size,
		Quality:   taskImage.Quality,
		N:         taskImage.N,
		Prompt:    taskImage.Prompt,
		Error:     taskImage.Error,
		Deleted:   true,
	}

	if taskImage.CompletedAt != 0 {
		response.CompletedAt = &taskImage.CompletedAt
	}

	if taskImage.ExpiresAt != 0 {
		response.ExpiresAt = &taskImage.ExpiresAt
	}

	return response, nil
}

// Content
func (s *sImage) Content(ctx context.Context, params *v1.ContentReq) (response smodel.ImageContentResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Content time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		if response.TotalTime == 0 {
			response.TotalTime = gtime.TimestampMilli() - now
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_CONTENT,
					ImageId:      params.ImageId,
					RequestData:  util.ConvToMap(params.ImageContentRequest),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskImages, err := dao.TaskImage.Find(ctx, bson.M{"image_id": params.ImageId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	taskImage := pickRepresentativeTaskImage(taskImages)
	if taskImage == nil {
		err = errors.NewError(404, "invalid_request_error", "Image with id '"+params.ImageId+"' not found.", "invalid_request_error", nil)
		logger.Error(ctx, err)
		return response, err
	}

	if taskImage.Status != "completed" {
		err = errors.NewError(404, "invalid_request_error", "Image is not ready yet, use GET /v1/images/generations/{image_id} to check status.", "invalid_request_error", nil)
		return response, err
	}

	mak.Model = taskImage.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if config.Cfg.ImageTask.IsEnableStorage {

		// 优先使用FilePaths按Index取对应文件, 兼容历史数据回退到FilePath
		var targetPath string
		if len(taskImage.FilePaths) > 0 {
			index := params.Index
			if index < 0 || index >= len(taskImage.FilePaths) {
				index = 0
			}
			targetPath = taskImage.FilePaths[index]
		} else if taskImage.FilePath != "" {
			targetPath = taskImage.FilePath
		}

		if targetPath != "" {
			if bytes := gfile.GetBytes(targetPath); bytes != nil {
				response = smodel.ImageContentResponse{Data: bytes}
				return response, nil
			}
		}
	}

	return response, nil
}

// 返回任务状态的排序权重, 数值越小优先级越高
func statusPriority(status string) int {
	switch status {
	case "completed":
		return 0
	case "in_progress":
		return 1
	case "queued":
		return 2
	case "failed":
		return 3
	case "expired":
		return 4
	case "deleted":
		return 5
	default:
		return 6
	}
}

// 返回同状态下的排序键, 越小越靠前
// 已完成按 completed_at 升序(为 0 视为异常, 排到最后), 其余状态按 created_at 升序
func taskImageOrderKey(t *entity.TaskImage) int64 {
	if t.Status == "completed" {
		if t.CompletedAt == 0 {
			return math.MaxInt64
		}
		return t.CompletedAt
	}
	return t.CreatedAt
}

// 从同一 image_id 的多条记录中选出代表记录
// 优先级: completed > in_progress > queued > failed > expired > deleted
// 同状态下: 已完成取最早完成的一条, 其余取最早创建的一条; 无记录时返回 nil
func pickRepresentativeTaskImage(taskImages []*entity.TaskImage) *entity.TaskImage {

	var best *entity.TaskImage

	for _, t := range taskImages {

		if best == nil {
			best = t
			continue
		}

		pt, pb := statusPriority(t.Status), statusPriority(best.Status)
		if pt < pb || (pt == pb && taskImageOrderKey(t) < taskImageOrderKey(best)) {
			best = t
		}
	}

	return best
}

// 为相对路径的图像URL添加StorageBaseUrl前缀
func buildStorageUrl(imageUrl string) string {
	if imageUrl == "" {
		return ""
	}
	if config.Cfg.ImageTask.StorageBaseUrl != "" {
		if gstr.HasSuffix(config.Cfg.ImageTask.StorageBaseUrl, "/") {
			imageUrl = gstr.TrimLeftStr(imageUrl, "/")
		} else if !gstr.HasPrefix(imageUrl, "/") {
			imageUrl = "/" + imageUrl
		}
	}
	return config.Cfg.ImageTask.StorageBaseUrl + imageUrl
}

// 同步转储图片到本地存储, 改写response中图片的访问地址, 返回与response.Data等长的文件路径列表及过期时间
func saveImageStorage(ctx context.Context, response *smodel.ImageResponse, outputFormat string) (filePaths []string, expiresAt int64) {

	if !config.Cfg.ImageStorage.Open || len(response.Data) == 0 {
		return nil, 0
	}

	storageDir := config.Cfg.ImageStorage.StorageDir
	if storageDir == "" {
		storageDir = "./resource/public/image/"
	} else if !gstr.HasSuffix(storageDir, "/") {
		storageDir = storageDir + "/"
	}

	if outputFormat == "" {
		outputFormat = "png"
	}

	traceId := gtrace.GetTraceID(ctx)
	filePaths = make([]string, len(response.Data))
	hasStored := false

	for i := range response.Data {

		imageData := response.Data[i]
		var imageBytes []byte

		if len(imageData.B64Json) > 0 {
			if decoded, err := base64.StdEncoding.DecodeString(imageData.B64Json); err == nil {
				imageBytes = decoded
			} else {
				logger.Error(ctx, err)
			}
		} else if len(imageData.Url) > 0 {

			if gstr.HasPrefix(imageData.Url, "data:") {

				if decoded, err := decodeDataURI(imageData.Url); err == nil {
					imageBytes = decoded
				} else {
					logger.Error(ctx, err)
				}

			} else if gstr.HasPrefix(imageData.Url, "http") {

				timeout := config.Cfg.ImageStorage.DownloadTimeout * time.Second
				if timeout <= 0 {
					timeout = config.Cfg.Base.ShortTimeout * time.Second
				}

				if body, _, err := downloadImage(ctx, imageData.Url, timeout); err != nil {
					logger.Error(ctx, err)
					response.Data[i].Url = ""
				} else {
					imageBytes = body
				}
			}
		}

		if imageBytes == nil {
			continue
		}

		fileName := fmt.Sprintf("%s%d.%s", traceId, i, outputFormat)

		if err := gfile.PutBytes(storageDir+fileName, imageBytes); err != nil {
			logger.Error(ctx, err)
			continue
		}

		var imageUrl string
		if gstr.HasPrefix(storageDir, "./resource/public/") {
			imageUrl = "/public/" + gstr.TrimLeftStr(storageDir, "./resource/public/") + fileName
		} else if config.Cfg.ImageStorage.StorageBaseUrl == "" {
			imageUrl = "/open/image/" + fileName
		} else {
			imageUrl = fileName
		}

		if config.Cfg.ImageStorage.StorageBaseUrl != "" {
			if gstr.HasSuffix(config.Cfg.ImageStorage.StorageBaseUrl, "/") {
				imageUrl = gstr.TrimLeftStr(imageUrl, "/")
			} else if !gstr.HasPrefix(imageUrl, "/") {
				imageUrl = "/" + imageUrl
			}
			imageUrl = config.Cfg.ImageStorage.StorageBaseUrl + imageUrl
		}

		response.Data[i].Url = imageUrl
		filePaths[i] = storageDir + fileName
		hasStored = true
	}

	if !hasStored {
		return nil, 0
	}

	if config.Cfg.ImageStorage.StorageExpiresAt > 0 {
		expiresAt = gtime.NewFromTimeStamp(gtime.TimestampMilli() / 1000).Add(config.Cfg.ImageStorage.StorageExpiresAt * time.Minute).Unix()
	}

	return filePaths, expiresAt
}

// 通过URL下载图像字节数据, 下载失败或内容为空时按配置重试
func downloadImage(ctx context.Context, imageUrl string, timeout time.Duration, retry ...int) ([]byte, http.Header, error) {

	body, respHeader, err := sutil.HttpGet(ctx, imageUrl, nil, nil, nil, timeout, config.Cfg.Http.ProxyUrl, nil)
	if err == nil && len(body) > 0 {
		return body, respHeader, nil
	}

	if err != nil {
		logger.Errorf(ctx, "sImage downloadImage url: %s, error: %v", imageUrl, err)
	} else {
		err = errors.New("downloaded image content is empty")
		logger.Errorf(ctx, "sImage downloadImage url: %s, empty content", imageUrl)
	}

	retryCount := config.Cfg.ImageStorage.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}

	if len(retry) == retryCount {
		return nil, nil, err
	}

	retry = append(retry, 1)

	time.Sleep(time.Duration(len(retry)*5) * time.Second)

	logger.Infof(ctx, "sImage downloadImage retry: %d, url: %s", len(retry), imageUrl)

	return downloadImage(ctx, imageUrl, timeout, retry...)
}

// 解析 data URI(形如 data:[<mediatype>][;base64],<data>), 返回解码后的字节数据
// 兼容任意图片格式(png/jpeg/webp/gif等)及可选的 charset、base64 标记, 不再硬编码具体 MIME 类型
func decodeDataURI(dataURI string) ([]byte, error) {

	// 必须以 data: 开头, 且包含分隔数据的逗号
	if !gstr.HasPrefix(dataURI, "data:") {
		return nil, errors.New("invalid data uri: missing data: prefix")
	}

	idx := gstr.Pos(dataURI, ",")
	if idx < 0 {
		return nil, errors.New("invalid data uri: missing comma separator")
	}

	meta := dataURI[len("data:"):idx]
	payload := dataURI[idx+1:]

	// meta 形如 image/png;base64 或 image/jpeg 或 ;base64, 末段为 base64 时按 base64 解码, 否则按 URL 编码解码
	if gstr.HasSuffix(gstr.ToLower(meta), "base64") {
		return base64.StdEncoding.DecodeString(payload)
	}

	// 非 base64 的 data URI 为百分号编码的原始数据
	decoded, err := url.QueryUnescape(payload)
	if err != nil {
		return nil, err
	}

	return []byte(decoded), nil
}
