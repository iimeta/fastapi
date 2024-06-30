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
		client     sdk.Chat
		reqModel   *model.Model
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
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
			CompletionTokens: reqModel.TextQuota.FixedQuota,
			TotalTokens:      reqModel.TextQuota.FixedQuota,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if retryInfo == nil && err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, usage.TotalTokens); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				reqModel.ModelAgent = modelAgent

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

				s.SaveChat(ctx, reqModel, realModel, fallbackModel, k, &params, imageRes, retryInfo)

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

// 保存文生图聊天数据
func (s *sImage) SaveChat(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, imageReq *sdkm.ImageRequest, imageRes *model.ImageRes, retryInfo *mcommon.Retry) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageRes.Error != nil && errors.Is(imageRes.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	completion := ""
	for i, data := range imageRes.Data {

		if len(completion) > 0 {
			completion += "\n\n"
		}

		completion += fmt.Sprintf("%d. %s", i+1, data.URL)
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Prompt:       imageReq.Prompt,
		Completion:   completion,
		ConnTime:     imageRes.ConnTime,
		Duration:     imageRes.Duration,
		TotalTime:    imageRes.TotalTime,
		InternalTime: imageRes.InternalTime,
		ReqTime:      imageRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(imageRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
	}

	chat.Corp = reqModel.Corp
	chat.ModelId = reqModel.Id
	chat.Name = reqModel.Name
	chat.Model = reqModel.Model
	chat.Type = reqModel.Type
	chat.ImageQuotas = reqModel.ImageQuotas

	chat.IsEnablePresetConfig = realModel.IsEnablePresetConfig
	chat.PresetConfig = realModel.PresetConfig
	chat.IsEnableForward = realModel.IsEnableForward
	chat.ForwardConfig = realModel.ForwardConfig
	chat.IsEnableModelAgent = realModel.IsEnableModelAgent
	chat.RealModelId = realModel.Id
	chat.RealModelName = realModel.Name
	chat.RealModel = realModel.Model

	chat.PromptTokens = imageRes.Usage.PromptTokens
	chat.CompletionTokens = imageRes.Usage.CompletionTokens
	chat.TotalTokens = imageRes.Usage.TotalTokens

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

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

	if key != nil {
		chat.Key = key.Key
	}

	if imageRes.Error != nil {
		chat.ErrMsg = imageRes.Error.Error()
		if common.IsAborted(imageRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if retryInfo != nil {

		chat.IsRetry = retryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if chat.IsRetry && imageRes.Error == nil {
			chat.Status = 3
			chat.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
