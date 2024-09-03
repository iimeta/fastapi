package image

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	sdk "github.com/iimeta/fastapi-sdk"
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
func (s *sImage) Generations(ctx context.Context, params sdkm.ImageRequest, fallbackModel *model.Model, retry ...int) (response sdkm.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Generations time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		client     sdk.Client
		reqModel   *model.Model
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
		imageQuota mcommon.ImageQuota
		key        string
		baseUrl    string
		path       string
		agentTotal int
		keyTotal   int
		retryInfo  *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: imageQuota.FixedQuota * len(response.Data),
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if retryInfo == nil && err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, usage.TotalTokens, k.Key); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				realModel.ModelAgent = modelAgent

				imageRes := &model.ImageRes{
					Created:      response.Created,
					Data:         response.Data,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && err == nil {
					imageRes.Usage = *usage
				}

				s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, imageRes, retryInfo)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
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
					return s.Generations(ctx, params, fallbackModel)
				}
			}

			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl
			path = modelAgent.Path

			if keyTotal, k, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				logger.Error(ctx, err)

				service.ModelAgent().RecordErrorModelAgent(ctx, realModel, modelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, modelAgent)
				}

				if realModel.IsEnableFallback {
					if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
						retryInfo = &mcommon.Retry{
							IsRetry:    true,
							RetryCount: len(retry),
							ErrMsg:     err.Error(),
						}
						return s.Generations(ctx, params, fallbackModel)
					}
				}

				return response, err
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
					return s.Generations(ctx, params, fallbackModel)
				}
			}

			return response, err
		}
	}

	request := params
	key = k.Key

	imageQuota = common.GetImageQuota(realModel, request.Size)
	request.Size = fmt.Sprintf("%dx%d", imageQuota.Width, imageQuota.Height)

	if !gstr.Contains(realModel.Model, "*") {
		request.Model = realModel.Model
	}

	client, err = common.NewClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.Generations(ctx, params, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.Image(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, realModel, k, modelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if realModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, k)
				} else {
					service.Key().DisabledModelKey(ctx, k)
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
						return s.Generations(ctx, params, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Generations(ctx, params, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sImage) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, imageReq *sdkm.ImageRequest, imageRes *model.ImageRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageRes.Error != nil && errors.Is(imageRes.Error, errors.ERR_MODEL_NOT_FOUND) {
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
		TotalTime:      imageRes.TotalTime,
		InternalTime:   imageRes.InternalTime,
		ReqTime:        imageRes.EnterTime,
		ReqDate:        gtime.NewFromTimeStamp(imageRes.EnterTime).Format("Y-m-d"),
		ClientIp:       g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:       g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:        util.GetLocalIp(),
		Status:         1,
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
		image.ImageQuotas = reqModel.ImageQuotas
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

		if image.IsEnableModelAgent && realModel.ModelAgent != nil {
			image.ModelAgentId = realModel.ModelAgent.Id
			image.ModelAgent = &do.ModelAgent{
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

	image.TotalTokens = imageRes.Usage.TotalTokens

	if fallbackModel != nil {
		image.IsEnableFallback = true
		image.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

	if key != nil {
		image.Key = key.Key
	}

	if imageRes.Error != nil {
		image.ErrMsg = imageRes.Error.Error()
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
		logger.Error(ctx, err)

		if len(retry) == 5 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sImage SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModel, key, imageReq, imageRes, retryInfo, retry...)
	}
}
