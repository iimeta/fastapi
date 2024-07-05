package midjourney

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sMidjourney struct{}

func init() {
	service.RegisterMidjourney(New())
}

func New() service.IMidjourney {
	return &sMidjourney{}
}

// Main
func (s *sMidjourney) Main(ctx context.Context, request *ghttp.Request, retry ...int) (res []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Main time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		m          *model.Model
		key        *model.Key
		modelAgent *model.ModelAgent
		baseUrl    = config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl
		keyTotal   int
		response   sdkm.MidjourneyResponse
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			CompletionTokens: m.TextQuota.FixedQuota,
			TotalTokens:      m.TextQuota.FixedQuota,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, usage.TotalTokens); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				midjourneyProxyResponse := model.MidjourneyResponse{
					MidjourneyResponse: response,
					TotalTime:          response.TotalTime,
					Error:              err,
					InternalTime:       internalTime,
					EnterTime:          enterTime,
				}

				if err == nil {
					midjourneyProxyResponse.Usage = *usage
				}

				s.SaveLog(ctx, m, key, request.GetBodyString(), midjourneyProxyResponse)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, "Midjourney", service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if m.IsEnableModelAgent {

		if _, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if keyTotal, key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, m, modelAgent)
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else {
		if keyTotal, key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	client := sdk.NewMidjourneyClient(ctx, baseUrl, request.RequestURI, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)
	response, err = client.Main(ctx, request.GetBody())
	if err != nil {
		logger.Error(ctx, err)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return nil, err
			} else if config.Cfg.Api.Retry < 0 && len(retry) == keyTotal {
				return nil, err
			} else if config.Cfg.Api.Retry == 0 {
				return nil, err
			}
		}

		return s.Main(ctx, request, append(retry, 1)...)
	}

	return response.Response, nil
}

// Fetch
func (s *sMidjourney) Fetch(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyFetchResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Fetch time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		m          *model.Model
		key        *model.Key
		modelAgent *model.ModelAgent
		baseUrl    = config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl
		keyTotal   int
	)

	defer func() {

		// 替换图片CDN地址
		if config.Cfg.Midjourney.CdnUrl != "" && config.Cfg.Midjourney.MidjourneyProxy.CdnOriginalUrl != "" {
			response.ImageUrl = gstr.Replace(response.ImageUrl, config.Cfg.Midjourney.MidjourneyProxy.CdnOriginalUrl, config.Cfg.Midjourney.CdnUrl)
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			CompletionTokens: m.TextQuota.FixedQuota,
			TotalTokens:      m.TextQuota.FixedQuota,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, usage.TotalTokens); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				midjourneyProxyResponse := model.MidjourneyResponse{
					MidjourneyResponse: sdkm.MidjourneyResponse{
						Response: []byte(fmt.Sprintf("taskId: %s\nimageUrl: %s", params.TaskId, response.ImageUrl)),
					},
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if err == nil {
					midjourneyProxyResponse.Usage = *usage
				}

				s.SaveLog(ctx, m, key, params.TaskId, midjourneyProxyResponse)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, "Midjourney", service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if m.IsEnableModelAgent {

		if _, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

			if keyTotal, key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, m, modelAgent)
				logger.Error(ctx, err)
				return response, err
			}
		}

	} else {
		if keyTotal, key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	client := sdk.NewMidjourneyClient(ctx, baseUrl, "/mj/task/${taskId}/fetch", key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)
	response, err = client.Fetch(ctx, params)
	if err != nil {
		logger.Error(ctx, err)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return response, err
			} else if config.Cfg.Api.Retry < 0 && len(retry) == keyTotal {
				return response, err
			} else if config.Cfg.Api.Retry == 0 {
				return response, err
			}
		}

		return s.Fetch(ctx, params, append(retry, 1)...)
	}

	return response, nil
}

// 保存Midjourney日志
func (s *sMidjourney) SaveLog(ctx context.Context, model *model.Model, key *model.Key, prompt string, response model.MidjourneyResponse) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if response.Error != nil && errors.Is(response.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Prompt:       prompt,
		Completion:   gconv.String(response.Response),
		ConnTime:     response.ConnTime,
		Duration:     response.Duration,
		TotalTime:    response.TotalTime,
		InternalTime: response.InternalTime,
		ReqTime:      response.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(response.EnterTime).Format("Y-m-d"),
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
		chat.ImageQuotas = model.ImageQuotas
		chat.IsEnableModelAgent = model.IsEnableModelAgent

		chat.PromptTokens = response.Usage.PromptTokens
		chat.CompletionTokens = response.Usage.CompletionTokens
		chat.TotalTokens = response.Usage.TotalTokens

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
	}

	if key != nil {
		chat.Key = key.Key
	}

	if response.Error != nil {
		chat.ErrMsg = response.Error.Error()
		chat.Status = -1
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
