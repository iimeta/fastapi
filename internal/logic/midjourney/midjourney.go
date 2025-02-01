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
func (s *sMidjourney) Submit(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Submit time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		defaultModel = "midjourney"
		mak          = &common.MAK{
			Model:              defaultModel,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		midjourneyQuota mcommon.MidjourneyQuota
		baseUrl         = config.Cfg.Midjourney.ApiBaseUrl
		path            = request.RequestURI[3:]
		retryInfo       *mcommon.Retry
		reqUrl          = request.RequestURI
		taskId          string
		prompt          = request.GetMapStrStr()["prompt"]
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: midjourneyQuota.FixedQuota,
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, mak.Key.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				mak.RealModel.ModelAgent = mak.ModelAgent

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

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, fallbackModelAgent, fallbackModel, mak.Key, midjourneyResponse, retryInfo)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if midjourneyQuota, err = common.GetMidjourneyQuota(mak.RealModel, request, path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	client := sdk.NewMidjourneyClient(ctx, baseUrl, midjourneyQuota.Path, mak.RealKey, config.Cfg.Midjourney.ApiSecretHeader, request.Method, config.Cfg.Http.ProxyUrl)

	response, err = client.Request(ctx, request.GetBody())
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
							return s.Submit(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Submit(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
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

			return s.Submit(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
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
func (s *sMidjourney) Task(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney Task time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		defaultModel = "midjourney"
		mak          = &common.MAK{
			Model:              defaultModel,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		midjourneyQuota mcommon.MidjourneyQuota
		baseUrl         = config.Cfg.Midjourney.ApiBaseUrl
		path            = request.RequestURI[3:]
		taskId          = request.GetRouterMap()["taskId"]
		imageUrl        string
		retryInfo       *mcommon.Retry
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: midjourneyQuota.FixedQuota,
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, usage.TotalTokens, mak.Key.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				mak.RealModel.ModelAgent = mak.ModelAgent

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

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, fallbackModelAgent, fallbackModel, mak.Key, midjourneyResponse, retryInfo)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if midjourneyQuota, err = common.GetMidjourneyQuota(mak.RealModel, request, path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	client := sdk.NewMidjourneyClient(ctx, baseUrl, path, mak.RealKey, config.Cfg.Midjourney.ApiSecretHeader, http.MethodGet, config.Cfg.Http.ProxyUrl)

	response, err = client.Request(ctx, request.GetBody())
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
							return s.Task(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Task(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
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

			return s.Task(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
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
	if config.Cfg.Midjourney.CdnUrl != "" && config.Cfg.Midjourney.CdnOriginalUrl != "" && imageUrl != "" {

		imageUrl = gstr.Replace(imageUrl, config.Cfg.Midjourney.CdnOriginalUrl, config.Cfg.Midjourney.CdnUrl)
		data["imageUrl"] = imageUrl

		if response.Response, err = gjson.Marshal(data); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	return response, nil
}

// 保存日志
func (s *sMidjourney) SaveLog(ctx context.Context, reqModel, realModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, response model.MidjourneyResponse, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if response.Error != nil && (errors.Is(response.Error, errors.ERR_MODEL_NOT_FOUND) || errors.Is(response.Error, errors.ERR_MODEL_DISABLED)) {
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
		Host:         g.RequestFromCtx(ctx).GetHost(),
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

	if fallbackModelAgent != nil {
		midjourney.IsEnableFallback = true
		midjourney.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		midjourney.IsEnableFallback = true
		if midjourney.FallbackConfig == nil {
			midjourney.FallbackConfig = new(mcommon.FallbackConfig)
		}
		midjourney.FallbackConfig.Model = fallbackModel.Model
		midjourney.FallbackConfig.ModelName = fallbackModel.Name
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

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sMidjourney SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModelAgent, fallbackModel, key, response, retryInfo, retry...)
	}
}
