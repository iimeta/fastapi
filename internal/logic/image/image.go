package image

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"github.com/iimeta/go-openai"
	"math"
	"slices"
	"time"
)

type sImage struct{}

func init() {
	service.RegisterImage(New())
}

func New() service.IImage {
	return &sImage{}
}

// Generations
func (s *sImage) Generations(ctx context.Context, params sdkm.ImageGenerationRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Generations time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client          sdk.Client
		generationQuota mcommon.GenerationQuota
		retryInfo       *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			if mak.ReqModel.ImageQuota.BillingMethod == 1 {

				usage.TotalTokens = generationQuota.FixedQuota * len(response.Data)

				if usage.InputTokens > 0 {
					usage.TotalTokens += int(math.Ceil(float64(usage.InputTokensDetails.TextTokens) * mak.ReqModel.ImageQuota.TextRatio))
					usage.TotalTokens += int(math.Ceil(float64(usage.InputTokensDetails.ImageTokens) * mak.ReqModel.ImageQuota.InputRatio))
				}

				if usage.OutputTokens > 0 {
					usage.TotalTokens += int(math.Ceil(float64(usage.OutputTokens) * mak.ReqModel.ImageQuota.OutputRatio))
				}

			} else {
				usage.TotalTokens = mak.ReqModel.ImageQuota.FixedQuota
			}

			// 分组折扣
			if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
				usage.TotalTokens = int(math.Ceil(float64(usage.TotalTokens) * mak.Group.Discount))
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, mak.Key.Key, mak.Group); err != nil {
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

				s.SaveLog(ctx, mak.Group, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, imageRes, retryInfo)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if mak.ReqModel.ImageQuota.BillingMethod == 1 {
		generationQuota = common.GetImageGenerationQuota(mak.RealModel, params.Quality, params.Size)
		params.Quality = generationQuota.Quality
		params.Size = fmt.Sprintf("%dx%d", generationQuota.Width, generationQuota.Height)
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

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.ImageGeneration(ctx, request)
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
							return s.Generations(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Generations(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Generations(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// Edits
func (s *sImage) Edits(ctx context.Context, params model.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ImageResponse, err error) {

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
		client          sdk.Client
		generationQuota mcommon.GenerationQuota
		retryInfo       *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := response.Usage

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			if mak.ReqModel.ImageQuota.BillingMethod == 1 {

				usage.TotalTokens = generationQuota.FixedQuota * len(response.Data)

				if usage.InputTokens > 0 {
					usage.TotalTokens += int(math.Ceil(float64(usage.InputTokensDetails.TextTokens) * mak.ReqModel.ImageQuota.TextRatio))
					usage.TotalTokens += int(math.Ceil(float64(usage.InputTokensDetails.ImageTokens) * mak.ReqModel.ImageQuota.InputRatio))
				}

				if usage.OutputTokens > 0 {
					usage.TotalTokens += int(math.Ceil(float64(usage.OutputTokens) * mak.ReqModel.ImageQuota.OutputRatio))
				}

			} else {
				usage.TotalTokens = mak.ReqModel.ImageQuota.FixedQuota
			}

			// 分组折扣
			if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
				usage.TotalTokens = int(math.Ceil(float64(usage.TotalTokens) * mak.Group.Discount))
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, mak.Key.Key, mak.Group); err != nil {
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

				imageReq := &sdkm.ImageGenerationRequest{
					Prompt:         params.Prompt,
					Background:     params.Background,
					Model:          params.Model,
					N:              params.N,
					Quality:        params.Quality,
					ResponseFormat: params.ResponseFormat,
					Size:           params.Size,
					User:           params.User,
				}

				s.SaveLog(ctx, mak.Group, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, imageReq, imageRes, retryInfo)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if mak.ReqModel.ImageQuota.BillingMethod == 1 {
		generationQuota = common.GetImageGenerationQuota(mak.RealModel, params.Quality, params.Size)
		params.Quality = generationQuota.Quality
		params.Size = fmt.Sprintf("%dx%d", generationQuota.Width, generationQuota.Height)
	}

	request := sdkm.ImageEditRequest{
		Prompt:         params.Prompt,
		Background:     params.Background,
		Model:          params.Model,
		N:              params.N,
		Quality:        params.Quality,
		ResponseFormat: params.ResponseFormat,
		Size:           params.Size,
		User:           params.User,
	}

	for _, image := range params.Image {
		request.Image = append(request.Image, image.FileHeader)
	}

	if params.Mask != nil {
		request.Mask = params.Mask.FileHeader
	}

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sImage Edits request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.ImageEdit(ctx, request)
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
func (s *sImage) SaveLog(ctx context.Context, group *model.Group, reqModel, realModel *model.Model, modelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, imageReq *sdkm.ImageGenerationRequest, imageRes *model.ImageRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageRes.Error != nil && (errors.Is(imageRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(imageRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(imageRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(imageRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(imageRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(imageRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	image := do.Image{
		TraceId:        gctx.CtxId(ctx),
		UserId:         service.Session().GetUserId(ctx),
		AppId:          service.Session().GetAppId(ctx),
		Prompt:         imageReq.Prompt,
		Size:           imageReq.Size,
		N:              imageReq.N,
		Quality:        imageReq.Quality,
		Style:          imageReq.Style,
		ResponseFormat: imageReq.ResponseFormat,
		InputTokens:    imageRes.Usage.InputTokens,
		OutputTokens:   imageRes.Usage.OutputTokens,
		TextTokens:     imageRes.Usage.InputTokensDetails.TextTokens,
		ImageTokens:    imageRes.Usage.InputTokensDetails.ImageTokens,
		TotalTokens:    imageRes.Usage.TotalTokens,
		TotalTime:      imageRes.TotalTime,
		InternalTime:   imageRes.InternalTime,
		ReqTime:        imageRes.EnterTime,
		ReqDate:        gtime.NewFromTimeStamp(imageRes.EnterTime).Format("Y-m-d"),
		ClientIp:       g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:       g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:        util.GetLocalIp(),
		Status:         1,
		Host:           g.RequestFromCtx(ctx).GetHost(),
		Rid:            service.Session().GetRid(ctx),
	}

	if group != nil {
		image.GroupId = group.Id
		image.GroupName = group.Name
		image.Discount = group.Discount
	}

	for _, data := range imageRes.Data {
		image.ImageData = append(image.ImageData, mcommon.ImageData{
			URL: data.URL,
			//B64JSON:       data.B64JSON, // 太大了, 不存
			RevisedPrompt: data.RevisedPrompt,
		})
	}

	if reqModel != nil {
		image.Corp = reqModel.Corp
		image.ModelId = reqModel.Id
		image.Name = reqModel.Name
		image.Model = reqModel.Model
		image.Type = reqModel.Type
		image.ImageQuota = reqModel.ImageQuota
	}

	if realModel != nil {
		image.IsEnablePresetConfig = realModel.IsEnablePresetConfig
		image.PresetConfig = realModel.PresetConfig
		image.IsEnableForward = realModel.IsEnableForward
		image.ForwardConfig = realModel.ForwardConfig
		image.IsEnableModelAgent = realModel.IsEnableModelAgent
		image.RealModelId = realModel.Id
		image.RealModelName = realModel.Name
		image.RealModel = realModel.Model
	}

	if image.IsEnableModelAgent && modelAgent != nil {
		image.ModelAgentId = modelAgent.Id
		image.ModelAgent = &do.ModelAgent{
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
		image.IsEnableFallback = true
		image.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		image.IsEnableFallback = true
		if image.FallbackConfig == nil {
			image.FallbackConfig = new(mcommon.FallbackConfig)
		}
		image.FallbackConfig.Model = fallbackModel.Model
		image.FallbackConfig.ModelName = fallbackModel.Name
	}

	if key != nil {
		image.Key = key.Key
	}

	if imageRes.Error != nil {

		image.ErrMsg = imageRes.Error.Error()
		openaiApiError := &openai.APIError{}
		if errors.As(imageRes.Error, &openaiApiError) {
			image.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(imageRes.Error) {
			image.Status = 2
		} else {
			image.Status = -1
		}
	}

	if retryInfo != nil {

		image.IsRetry = retryInfo.IsRetry
		image.Retry = &mcommon.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if image.IsRetry {
			image.Status = 3
			image.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Image.Insert(ctx, image); err != nil {
		logger.Errorf(ctx, "sImage SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			imageReq.Prompt = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sImage SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, group, reqModel, realModel, modelAgent, fallbackModelAgent, fallbackModel, key, imageReq, imageRes, retryInfo, retry...)
	}
}
