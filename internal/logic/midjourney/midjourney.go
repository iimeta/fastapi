package midjourney

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
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

// Imagine
func (s *sMidjourney) Imagine(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyResponse, err error) {

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
			TotalTokens:      200,
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

				midjourneyProxyResponse := model.MidjourneyProxyResponse{
					MidjourneyProxyResponse: response,
					Usage:                   usage,
					TotalTime:               response.TotalTime,
					Error:                   err,
					InternalTime:            internalTime,
					EnterTime:               enterTime,
				}

				s.SaveChat(ctx, m, key, params, midjourneyProxyResponse)

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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

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

// Change
func (s *sMidjourney) Change(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyResponse, err error) {

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
			TotalTokens:      200,
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

				midjourneyProxyResponse := model.MidjourneyProxyResponse{
					MidjourneyProxyResponse: response,
					Usage:                   usage,
					TotalTime:               response.TotalTime,
					Error:                   err,
					InternalTime:            internalTime,
					EnterTime:               enterTime,
				}

				s.SaveChat(ctx, m, key, params, midjourneyProxyResponse)

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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	if response, err = sdk.Change(ctx, midjourneyProxy, params); err != nil {
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

		return s.Change(ctx, params, append(retry, 1)...)
	}

	return response, nil
}

// Describe
func (s *sMidjourney) Describe(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyResponse, err error) {

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
			TotalTokens:      200,
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

				midjourneyProxyResponse := model.MidjourneyProxyResponse{
					MidjourneyProxyResponse: response,
					Usage:                   usage,
					TotalTime:               response.TotalTime,
					Error:                   err,
					InternalTime:            internalTime,
					EnterTime:               enterTime,
				}

				s.SaveChat(ctx, m, key, params, midjourneyProxyResponse)

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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	if response, err = sdk.Describe(ctx, midjourneyProxy, params); err != nil {
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

		return s.Describe(ctx, params, append(retry, 1)...)
	}

	return response, nil
}

// Blend
func (s *sMidjourney) Blend(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyResponse, err error) {

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
			TotalTokens:      200,
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

				midjourneyProxyResponse := model.MidjourneyProxyResponse{
					MidjourneyProxyResponse: response,
					Usage:                   usage,
					TotalTime:               response.TotalTime,
					Error:                   err,
					InternalTime:            internalTime,
					EnterTime:               enterTime,
				}

				s.SaveChat(ctx, m, key, params, midjourneyProxyResponse)

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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	if response, err = sdk.Blend(ctx, midjourneyProxy, params); err != nil {
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

		return s.Blend(ctx, params, append(retry, 1)...)
	}

	return response, nil
}

// Fetch
func (s *sMidjourney) Fetch(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyFetchResponse, err error) {

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
			TotalTokens:      200,
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

				midjourneyProxyResponse := model.MidjourneyProxyResponse{
					MidjourneyProxyResponse: sdkm.MidjourneyProxyResponse{
						Result: fmt.Sprintf("taskId: %s\nimageUrl: %s", params.TaskId, response.ImageUrl),
					},
					Usage:        usage,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				s.SaveChat(ctx, m, key, params, midjourneyProxyResponse)

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

	midjourneyProxy := sdk.NewMidjourneyProxy(ctx, baseUrl, key.Key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)

	if response, err = sdk.Fetch(ctx, midjourneyProxy, params); err != nil {
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

// 保存Midjourney数据
func (s *sMidjourney) SaveChat(ctx context.Context, model *model.Model, key *model.Key, request sdkm.MidjourneyProxyRequest, response model.MidjourneyProxyResponse) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Prompt:       request.Prompt,
		Completion:   response.Result,
		ConnTime:     response.ConnTime,
		Duration:     response.Duration,
		TotalTime:    response.TotalTime,
		InternalTime: response.InternalTime,
		ReqTime:      response.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(response.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		Status:       1,
	}

	if model != nil {
		chat.Corp = model.Corp
		chat.ModelId = model.Id
		chat.Name = model.Name
		chat.Model = model.Model
		chat.Type = model.Type
		chat.PromptRatio = model.PromptRatio
		chat.CompletionRatio = model.CompletionRatio
		chat.IsEnableModelAgent = model.IsEnableModelAgent
		if chat.IsEnableModelAgent {
			chat.ModelAgentId = model.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
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

	if response.Usage.TotalTokens != 0 {
		chat.PromptTokens = int(chat.PromptRatio * float64(response.Usage.PromptTokens))
		chat.CompletionTokens = int(chat.CompletionRatio * float64(response.Usage.CompletionTokens))
		chat.TotalTokens = chat.PromptTokens + chat.CompletionTokens
	}

	if response.Error != nil {
		chat.ErrMsg = response.Error.Error()
		chat.Status = -1
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
