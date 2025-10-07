package image

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	serrors "github.com/iimeta/fastapi-sdk/errors"
	smodel "github.com/iimeta/fastapi-sdk/model"
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
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			billingData := &mcommon.BillingData{
				ImageGenerationRequest: params,
				Usage:                  &usage,
			}

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			response.Usage = *billingData.Usage

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, spend.TotalSpendTokens, mak.Key.Key, mak.Group); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageRes := &model.ImageRes{
					Created:      response.Created,
					Data:         response.Data,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && (err == nil || common.IsAborted(err)) {
					imageRes.Usage = usage
				}

				s.SaveLog(ctx, model.ImageLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					ImageReq:           &params,
					ImageRes:           imageRes,
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
		return response, err
	}

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageGenerationRequest: params,
		}

		// 花费
		spend = common.Spend(ctx, mak, billingData, "image_generation")

		if spend.ImageGeneration.Pricing.Quality != "" {
			params.Quality = spend.ImageGeneration.Pricing.Quality
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

	response, err = common.NewAdapter(ctx, mak, false).ImageGenerations(ctx, data)
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
							return s.Generations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
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

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Generations(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
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
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			billingData := &mcommon.BillingData{
				Usage: &usage,
			}

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			response.Usage = *billingData.Usage

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, spend.TotalSpendTokens, mak.Key.Key, mak.Group); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				imageRes := &model.ImageRes{
					Created:      response.Created,
					Data:         response.Data,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && (err == nil || common.IsAborted(err)) {
					imageRes.Usage = usage
				}

				imageReq := &smodel.ImageGenerationRequest{
					Prompt:         params.Prompt,
					Background:     params.Background,
					Model:          params.Model,
					N:              params.N,
					Quality:        params.Quality,
					ResponseFormat: params.ResponseFormat,
					Size:           params.Size,
					User:           params.User,
				}

				s.SaveLog(ctx, model.ImageLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					ImageReq:           imageReq,
					ImageRes:           imageRes,
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
		return response, err
	}

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "image_generation") {

		billingData := &mcommon.BillingData{
			ImageEditRequest: params,
		}

		// 花费
		spend = common.Spend(ctx, mak, billingData, "image_generation")

		if spend.ImageGeneration.Pricing.Quality != "" {
			params.Quality = spend.ImageGeneration.Pricing.Quality
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

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Edits(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
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

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Edits(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sImage) SaveLog(ctx context.Context, imageLog model.ImageLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageLog.ImageRes.Error != nil && (errors.Is(imageLog.ImageRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(imageLog.ImageRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(imageLog.ImageRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(imageLog.ImageRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(imageLog.ImageRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(imageLog.ImageRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	image := do.Image{
		TraceId:        gctx.CtxId(ctx),
		UserId:         service.Session().GetUserId(ctx),
		AppId:          service.Session().GetAppId(ctx),
		Prompt:         imageLog.ImageReq.Prompt,
		Size:           imageLog.ImageReq.Size,
		N:              imageLog.ImageReq.N,
		Quality:        imageLog.ImageReq.Quality,
		Style:          imageLog.ImageReq.Style,
		ResponseFormat: imageLog.ImageReq.ResponseFormat,
		InputTokens:    imageLog.ImageRes.Usage.InputTokens,
		OutputTokens:   imageLog.ImageRes.Usage.OutputTokens,
		TextTokens:     imageLog.ImageRes.Usage.InputTokensDetails.TextTokens,
		ImageTokens:    imageLog.ImageRes.Usage.InputTokensDetails.ImageTokens,
		Spend:          imageLog.Spend,
		TotalTime:      imageLog.ImageRes.TotalTime,
		InternalTime:   imageLog.ImageRes.InternalTime,
		ReqTime:        imageLog.ImageRes.EnterTime,
		ReqDate:        gtime.NewFromTimeStamp(imageLog.ImageRes.EnterTime).Format("Y-m-d"),
		ClientIp:       g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:       g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:        util.GetLocalIp(),
		Status:         1,
		Host:           g.RequestFromCtx(ctx).GetHost(),
		Rid:            service.Session().GetRid(ctx),
	}

	for _, data := range imageLog.ImageRes.Data {
		image.ImageData = append(image.ImageData, mcommon.ImageData{
			Url: data.Url,
			//B64Json:       data.B64Json, // todo 太大了, 不存
			RevisedPrompt: data.RevisedPrompt,
		})
	}

	if imageLog.ReqModel != nil {
		image.ProviderId = imageLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, imageLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			image.ProviderName = provider.Name
		}
		image.ModelId = imageLog.ReqModel.Id
		image.ModelName = imageLog.ReqModel.Name
		image.Model = imageLog.ReqModel.Model
		image.ModelType = imageLog.ReqModel.Type
	}

	if imageLog.RealModel != nil {
		image.IsEnablePresetConfig = imageLog.RealModel.IsEnablePresetConfig
		image.PresetConfig = imageLog.RealModel.PresetConfig
		image.IsEnableForward = imageLog.RealModel.IsEnableForward
		image.ForwardConfig = imageLog.RealModel.ForwardConfig
		image.IsEnableModelAgent = imageLog.RealModel.IsEnableModelAgent
		image.RealModelId = imageLog.RealModel.Id
		image.RealModelName = imageLog.RealModel.Name
		image.RealModel = imageLog.RealModel.Model
	}

	if image.IsEnableModelAgent && imageLog.ModelAgent != nil {
		image.ModelAgentId = imageLog.ModelAgent.Id
		image.ModelAgent = &do.ModelAgent{
			ProviderId: imageLog.ModelAgent.ProviderId,
			Name:       imageLog.ModelAgent.Name,
			BaseUrl:    imageLog.ModelAgent.BaseUrl,
			Path:       imageLog.ModelAgent.Path,
			Weight:     imageLog.ModelAgent.Weight,
			Remark:     imageLog.ModelAgent.Remark,
		}
	}

	if imageLog.FallbackModelAgent != nil {
		image.IsEnableFallback = true
		image.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     imageLog.FallbackModelAgent.Id,
			ModelAgentName: imageLog.FallbackModelAgent.Name,
		}
	}

	if imageLog.FallbackModel != nil {
		image.IsEnableFallback = true
		if image.FallbackConfig == nil {
			image.FallbackConfig = new(mcommon.FallbackConfig)
		}
		image.FallbackConfig.Model = imageLog.FallbackModel.Model
		image.FallbackConfig.ModelName = imageLog.FallbackModel.Name
	}

	if imageLog.Key != nil {
		image.Key = imageLog.Key.Key
	}

	if imageLog.ImageRes.Error != nil {

		image.ErrMsg = imageLog.ImageRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(imageLog.ImageRes.Error, &openaiApiError) {
			image.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(imageLog.ImageRes.Error) {
			image.Status = 2
		} else {
			image.Status = -1
		}
	}

	if imageLog.RetryInfo != nil {

		image.IsRetry = imageLog.RetryInfo.IsRetry
		image.Retry = &mcommon.Retry{
			IsRetry:    imageLog.RetryInfo.IsRetry,
			RetryCount: imageLog.RetryInfo.RetryCount,
			ErrMsg:     imageLog.RetryInfo.ErrMsg,
		}

		if image.IsRetry {
			image.Status = 3
			image.ErrMsg = imageLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Image.Insert(ctx, image); err != nil {
		logger.Errorf(ctx, "sImage SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			imageLog.ImageReq.Prompt = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sImage SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, imageLog, retry...)
	}
}
