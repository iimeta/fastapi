package image

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
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
func (s *sImage) Generations(ctx context.Context, params sdkm.ImageRequest, retry ...int) (response sdkm.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Generations time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		reqModel   *model.Model
		realModel  = new(model.Model)
		k          *model.Key
		modelAgent *model.ModelAgent
		key        string
		baseUrl    string
		path       string
		agentTotal int
		keyTotal   int
		retryInfo  *do.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: reqModel.FixedQuota,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if retryInfo == nil && err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, reqModel, usage); err != nil {
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
					imageRes.Usage = usage
				}

				s.SaveChat(ctx, reqModel, k, &params, imageRes, retryInfo)

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

	*realModel = *reqModel
	baseUrl = realModel.BaseUrl
	path = realModel.Path

	if reqModel.IsEnableModelAgent {

		if agentTotal, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, reqModel); err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl
			path = modelAgent.Path

			if keyTotal, k, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, reqModel, modelAgent)
				logger.Error(ctx, err)
				return response, err
			}
		}

	} else {
		if keyTotal, k, err = service.Key().PickModelKey(ctx, reqModel); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	request := params
	request.Model = realModel.Model
	key = k.Key

	client := sdk.NewClient(ctx, getCorpCode(ctx, realModel.Corp), realModel.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	response, err = client.Image(ctx, request)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, reqModel, k, modelAgent)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return response, err
			} else if config.Cfg.Api.Retry < 0 {
				if realModel.IsEnableModelAgent {
					if len(retry) == agentTotal {
						return response, err
					}
				} else if len(retry) == keyTotal {
					return response, err
				}
			} else if config.Cfg.Api.Retry == 0 {
				return response, err
			}
		}

		isRetry, isDisabled := isNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if reqModel.IsEnableModelAgent {
					service.ModelAgent().DisabledModelAgentKey(ctx, k)
				} else {
					service.Key().DisabledModelKey(ctx, k)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {
			retryInfo = &do.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}
			return s.Generations(ctx, params, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存文生图聊天数据
func (s *sImage) SaveChat(ctx context.Context, model *model.Model, key *model.Key, imageReq *sdkm.ImageRequest, imageRes *model.ImageRes, retryInfo *do.Retry) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

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

	if model != nil {

		chat.Corp = model.Corp
		chat.ModelId = model.Id
		chat.Name = model.Name
		chat.Model = model.Model
		chat.Type = model.Type
		chat.BillingMethod = model.BillingMethod
		chat.PromptRatio = model.PromptRatio
		chat.CompletionRatio = model.CompletionRatio
		chat.FixedQuota = model.FixedQuota
		chat.IsEnableModelAgent = model.IsEnableModelAgent
		if chat.IsEnableModelAgent && model.ModelAgent != nil {
			chat.ModelAgentId = model.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
				Corp:    model.ModelAgent.Corp,
				Name:    model.ModelAgent.Name,
				BaseUrl: model.ModelAgent.BaseUrl,
				Path:    model.ModelAgent.Path,
				Weight:  model.ModelAgent.Weight,
				Remark:  model.ModelAgent.Remark,
				Status:  model.ModelAgent.Status,
			}
		}

		if imageRes.Error == nil {
			chat.TotalTokens = imageRes.Usage.TotalTokens
		}
	}

	if key != nil {
		chat.Key = key.Key
	}

	if imageRes.Error != nil {
		chat.ErrMsg = imageRes.Error.Error()
		chat.Status = -1
	}

	if retryInfo != nil {

		chat.IsRetry = retryInfo.IsRetry
		chat.Retry = &do.Retry{
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

func getCorpCode(ctx context.Context, corpId string) string {

	corp, err := service.Corp().GetCacheCorp(ctx, corpId)
	if err != nil || corp == nil {
		corp, err = service.Corp().GetCorpAndSaveCache(ctx, corpId)
	}

	if corp != nil {
		return corp.Code
	}

	return corpId
}

func isNeedRetry(err error) (isRetry bool, isDisabled bool) {

	apiError := &sdkerr.ApiError{}
	if errors.As(err, &apiError) {

		switch apiError.HttpStatusCode {
		case 400:
			if errors.Is(err, sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED) {
				return false, false
			}
		case 401, 429:
			if errors.Is(err, sdkerr.ERR_INVALID_API_KEY) || errors.Is(err, sdkerr.ERR_INSUFFICIENT_QUOTA) {
				return true, true
			}
		}

		return true, false
	}

	reqError := &sdkerr.RequestError{}
	if errors.As(err, &reqError) {
		return true, false
	}

	return false, false
}
