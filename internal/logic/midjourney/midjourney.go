package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/sashabaranov/go-openai"
)

type sMidjourney struct{}

func init() {
	service.RegisterMidjourney(New())
}

func New() service.IMidjourney {
	return &sMidjourney{}
}

func (s *sMidjourney) Imagine(ctx context.Context, params sdkm.MidjourneyProxyImagineReq, retry ...int) (response sdkm.MidjourneyProxyImagineRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Imagine time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl = config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl
	var keyTotal int

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := openai.Usage{
			PromptTokens:     100,
			CompletionTokens: 100,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, m, usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				imageRes := &model.ImageRes{
					Usage:        usage,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				service.Image().SaveImage(ctx, m, key, &openai.ImageRequest{
					Prompt: params.Prompt,
				}, imageRes)

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

		if modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, config.Cfg.Midjourney.MidjourneyProxy.ApiSecret, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	if response, err = sdk.Imagine(ctx, midjourneyProxy, params); err != nil {
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

		return s.Imagine(ctx, params, append(retry, 1)...)
	}

	return response, nil
}

func (s *sMidjourney) Change(ctx context.Context, params sdkm.MidjourneyProxyChangeReq) (sdkm.MidjourneyProxyChangeRes, error) {

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl, config.Cfg.Midjourney.MidjourneyProxy.ApiSecret, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	return sdk.Change(ctx, midjourneyProxy, params)
}

func (s *sMidjourney) Describe(ctx context.Context, params sdkm.MidjourneyProxyDescribeReq) (sdkm.MidjourneyProxyDescribeRes, error) {

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl, config.Cfg.Midjourney.MidjourneyProxy.ApiSecret, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	return sdk.Describe(ctx, midjourneyProxy, params)
}

func (s *sMidjourney) Blend(ctx context.Context, params sdkm.MidjourneyProxyBlendReq) (sdkm.MidjourneyProxyBlendRes, error) {

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl, config.Cfg.Midjourney.MidjourneyProxy.ApiSecret, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	return sdk.Blend(ctx, midjourneyProxy, params)
}

func (s *sMidjourney) Fetch(ctx context.Context, taskId string) (sdkm.MidjourneyProxyFetchRes, error) {

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl, config.Cfg.Midjourney.MidjourneyProxy.ApiSecret, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	return sdk.Fetch(ctx, midjourneyProxy, taskId)
}
