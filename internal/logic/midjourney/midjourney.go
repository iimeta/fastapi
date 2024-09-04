package midjourney

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"net/http"
	"time"
)

type sMidjourney struct{}

func init() {
	service.RegisterMidjourney(New())
}

func New() service.IMidjourney {
	return &sMidjourney{}
}

// 任务提交
func (s *sMidjourney) Submit(ctx context.Context, request *ghttp.Request, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Submit time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		defaultModel    = "midjourney"
		reqModel        *model.Model
		realModel       = new(model.Model)
		k               *model.Key
		modelAgent      *model.ModelAgent
		midjourneyQuota mcommon.MidjourneyQuota
		key             string
		baseUrl         = config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl
		path            = request.RequestURI[3:]
		agentTotal      int
		keyTotal        int
		retryInfo       *mcommon.Retry
		reqUrl          = request.RequestURI
		taskId          string
		prompt          = request.GetMapStrStr()["prompt"]
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: midjourneyQuota.FixedQuota,
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

			realModel.ModelAgent = modelAgent

			midjourneyResponse := model.MidjourneyResponse{
				ReqUrl:             reqUrl,
				TaskId:             taskId,
				Prompt:             prompt,
				MidjourneyResponse: response,
				TotalTime:          response.TotalTime,
				Error:              err,
				InternalTime:       internalTime,
				EnterTime:          enterTime,
			}

			if err == nil {
				midjourneyResponse.Usage = *usage
			}

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, midjourneyResponse, retryInfo)

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, defaultModel, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
	}

	midjourneyQuota, err = common.GetMidjourneyQuota(realModel, request, path)
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

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
					return s.Submit(ctx, request, fallbackModel)
				}
			}

			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl

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
						return s.Submit(ctx, request, fallbackModel)
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
					return s.Submit(ctx, request, fallbackModel)
				}
			}

			return response, err
		}
	}

	key = k.Key

	client := sdk.NewMidjourneyClient(ctx, baseUrl, midjourneyQuota.Path, key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader, request.Method, config.Cfg.Http.ProxyUrl)
	response, err = client.Request(ctx, request.GetBody())
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
						return s.Submit(ctx, request, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Submit(ctx, request, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	data := map[string]interface{}{}
	if err = gjson.Unmarshal(response.Response, &data); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	taskId = data["result"].(string)

	return response, nil
}

// 任务查询
func (s *sMidjourney) Task(ctx context.Context, request *ghttp.Request, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Task time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		defaultModel    = "midjourney"
		reqModel        *model.Model
		realModel       = new(model.Model)
		k               *model.Key
		modelAgent      *model.ModelAgent
		midjourneyQuota mcommon.MidjourneyQuota
		key             string
		baseUrl         = config.Cfg.Midjourney.MidjourneyProxy.ApiBaseUrl
		path            = request.RequestURI[3:]
		agentTotal      int
		keyTotal        int
		taskId          = request.GetRouterMap()["taskId"]
		imageUrl        string
		retryInfo       *mcommon.Retry
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: midjourneyQuota.FixedQuota,
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

			realModel.ModelAgent = modelAgent

			midjourneyResponse := model.MidjourneyResponse{
				MidjourneyResponse: sdkm.MidjourneyResponse{
					Response: []byte(fmt.Sprintf("taskId: %s\nimageUrl: %s", taskId, imageUrl)),
				},
				TotalTime:    response.TotalTime,
				Error:        err,
				InternalTime: internalTime,
				EnterTime:    enterTime,
			}

			if retryInfo == nil && (err == nil || common.IsAborted(err)) {
				midjourneyResponse.Usage = *usage
			}

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, midjourneyResponse, retryInfo)

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, defaultModel, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if fallbackModel != nil {
		*realModel = *fallbackModel
	} else {
		*realModel = *reqModel
	}

	midjourneyQuota, err = common.GetMidjourneyQuota(realModel, request, path)
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

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
					return s.Task(ctx, request, fallbackModel)
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
						return s.Task(ctx, request, fallbackModel)
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
					return s.Task(ctx, request, fallbackModel)
				}
			}

			return response, err
		}
	}

	key = k.Key

	client := sdk.NewMidjourneyClient(ctx, baseUrl, path, key, config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader, http.MethodGet, config.Cfg.Http.ProxyUrl)
	response, err = client.Request(ctx, request.GetBody())
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
						return s.Task(ctx, request, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Task(ctx, request, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	data := map[string]interface{}{}
	if err = gjson.Unmarshal(response.Response, &data); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	imageUrl = data["imageUrl"].(string)

	// 替换图片CDN地址
	if config.Cfg.Midjourney.CdnUrl != "" && config.Cfg.Midjourney.MidjourneyProxy.CdnOriginalUrl != "" && imageUrl != "" {

		imageUrl = gstr.Replace(imageUrl, config.Cfg.Midjourney.MidjourneyProxy.CdnOriginalUrl, config.Cfg.Midjourney.CdnUrl)
		data["imageUrl"] = imageUrl

		if response.Response, err = gjson.Marshal(data); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	return response, nil
}

// 保存日志
func (s *sMidjourney) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, response model.MidjourneyResponse, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if response.Error != nil && errors.Is(response.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	midjourney := do.Midjourney{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		ReqUrl:       response.ReqUrl,
		TaskId:       response.TaskId,
		Action:       response.Action,
		Prompt:       response.Prompt,
		PromptEn:     response.PromptEn,
		ImageUrl:     response.ImageUrl,
		Progress:     response.Progress,
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

	if reqModel != nil {
		midjourney.Corp = reqModel.Corp
		midjourney.ModelId = reqModel.Id
		midjourney.Name = reqModel.Name
		midjourney.Model = reqModel.Model
		midjourney.Type = reqModel.Type
		midjourney.MidjourneyQuotas = reqModel.MidjourneyQuotas
	}

	if realModel != nil {

		midjourney.IsEnablePresetConfig = realModel.IsEnablePresetConfig
		midjourney.PresetConfig = realModel.PresetConfig
		midjourney.IsEnableForward = realModel.IsEnableForward
		midjourney.ForwardConfig = realModel.ForwardConfig
		midjourney.IsEnableModelAgent = realModel.IsEnableModelAgent
		midjourney.RealModelId = realModel.Id
		midjourney.RealModelName = realModel.Name
		midjourney.RealModel = realModel.Model

		if midjourney.IsEnableModelAgent && realModel.ModelAgent != nil {
			midjourney.ModelAgentId = realModel.ModelAgent.Id
			midjourney.ModelAgent = &do.ModelAgent{
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

	midjourney.TotalTokens = response.Usage.TotalTokens

	if fallbackModel != nil {
		midjourney.IsEnableFallback = true
		midjourney.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

	if key != nil {
		midjourney.Key = key.Key
	}

	if response.Response != nil {
		if err := gjson.Unmarshal(response.Response, &midjourney.Response); err != nil {
			logger.Error(ctx, err)
		}
	}

	if response.Error != nil {
		midjourney.ErrMsg = response.Error.Error()
		if common.IsAborted(response.Error) {
			midjourney.Status = 2
		} else {
			midjourney.Status = -1
		}
	}

	if retryInfo != nil {

		midjourney.IsRetry = retryInfo.IsRetry
		midjourney.Retry = &mcommon.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if midjourney.IsRetry {
			midjourney.Status = 3
			midjourney.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Midjourney.Insert(ctx, midjourney); err != nil {
		logger.Error(ctx, err)

		if len(retry) == 5 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sMidjourney SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModel, key, response, retryInfo, retry...)
	}
}
