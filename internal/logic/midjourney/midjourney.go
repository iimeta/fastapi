package midjourney

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	smodel "github.com/iimeta/fastapi-sdk/model"
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
)

type sMidjourney struct{}

func init() {
	service.RegisterMidjourney(New())
}

func New() service.IMidjourney {
	return &sMidjourney{}
}

// 任务提交
func (s *sMidjourney) Submit(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.MidjourneyResponse, err error) {

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
		baseUrl   = config.Cfg.Midjourney.ApiBaseUrl
		path      = request.RequestURI[3:]
		reqUrl    = request.RequestURI
		taskId    string
		prompt    = request.GetMapStrStr()["prompt"]
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &smodel.Usage{}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			billingData := &mcommon.BillingData{
				Path:  path,
				Usage: usage,
			}

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			usage = billingData.Usage

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

				s.SaveLog(ctx, model.MidjourneyLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					Response:           midjourneyResponse,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "midjourney") {

		billingData := &mcommon.BillingData{
			Path: path,
		}

		// 花费
		spend = common.Spend(ctx, mak, billingData, "midjourney")
		if spend.Midjourney.Pricing.Path == "" {
			return response, errors.ERR_PATH_NOT_FOUND
		}
	}

	client := sdk.NewMidjourneyClient(ctx, baseUrl, path, mak.RealKey, config.Cfg.Midjourney.ApiSecretHeader, request.Method, config.Cfg.Http.ProxyUrl)

	response, err = client.Request(ctx, request.GetBody())
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
func (s *sMidjourney) Task(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.MidjourneyResponse, err error) {

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
		baseUrl   = config.Cfg.Midjourney.ApiBaseUrl
		path      = request.RequestURI[3:]
		taskId    = request.GetRouterMap()["taskId"]
		imageUrl  string
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &smodel.Usage{}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			billingData := &mcommon.BillingData{
				Path:  path,
				Usage: usage,
			}

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			usage = billingData.Usage

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

				midjourneyResponse := model.MidjourneyResponse{
					MidjourneyResponse: smodel.MidjourneyResponse{
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

				s.SaveLog(ctx, model.MidjourneyLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					Response:           midjourneyResponse,
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

	if slices.Contains(mak.ReqModel.Pricing.BillingItems, "midjourney") {

		billingData := &mcommon.BillingData{
			Path: path,
		}

		// 花费
		spend = common.Spend(ctx, mak, billingData, "midjourney")
		if spend.Midjourney.Pricing.Path == "" {
			return response, errors.ERR_PATH_NOT_FOUND
		}
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
func (s *sMidjourney) SaveLog(ctx context.Context, midjourneyLog model.MidjourneyLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMidjourney SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if midjourneyLog.Response.Error != nil && (errors.Is(midjourneyLog.Response.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(midjourneyLog.Response.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(midjourneyLog.Response.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(midjourneyLog.Response.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(midjourneyLog.Response.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(midjourneyLog.Response.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	midjourney := do.Midjourney{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		ReqUrl:       midjourneyLog.Response.ReqUrl,
		TaskId:       midjourneyLog.Response.TaskId,
		Action:       midjourneyLog.Response.Action,
		Prompt:       midjourneyLog.Response.Prompt,
		PromptEn:     midjourneyLog.Response.PromptEn,
		ImageUrl:     midjourneyLog.Response.ImageUrl,
		Progress:     midjourneyLog.Response.Progress,
		Spend:        midjourneyLog.Spend,
		ConnTime:     midjourneyLog.Response.ConnTime,
		Duration:     midjourneyLog.Response.Duration,
		TotalTime:    midjourneyLog.Response.TotalTime,
		InternalTime: midjourneyLog.Response.InternalTime,
		ReqTime:      midjourneyLog.Response.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(midjourneyLog.Response.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if midjourneyLog.ReqModel != nil {
		midjourney.ProviderId = midjourneyLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, midjourneyLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			midjourney.ProviderName = provider.Name
		}
		midjourney.ModelId = midjourneyLog.ReqModel.Id
		midjourney.ModelName = midjourneyLog.ReqModel.Name
		midjourney.Model = midjourneyLog.ReqModel.Model
		midjourney.ModelType = midjourneyLog.ReqModel.Type
	}

	if midjourneyLog.RealModel != nil {
		midjourney.IsEnablePresetConfig = midjourneyLog.RealModel.IsEnablePresetConfig
		midjourney.PresetConfig = midjourneyLog.RealModel.PresetConfig
		midjourney.IsEnableForward = midjourneyLog.RealModel.IsEnableForward
		midjourney.ForwardConfig = midjourneyLog.RealModel.ForwardConfig
		midjourney.IsEnableModelAgent = midjourneyLog.RealModel.IsEnableModelAgent
		midjourney.RealModelId = midjourneyLog.RealModel.Id
		midjourney.RealModelName = midjourneyLog.RealModel.Name
		midjourney.RealModel = midjourneyLog.RealModel.Model
	}

	if midjourney.IsEnableModelAgent && midjourneyLog.ModelAgent != nil {
		midjourney.ModelAgentId = midjourneyLog.ModelAgent.Id
		midjourney.ModelAgent = &do.ModelAgent{
			ProviderId: midjourneyLog.ModelAgent.ProviderId,
			Name:       midjourneyLog.ModelAgent.Name,
			BaseUrl:    midjourneyLog.ModelAgent.BaseUrl,
			Path:       midjourneyLog.ModelAgent.Path,
			Weight:     midjourneyLog.ModelAgent.Weight,
			Remark:     midjourneyLog.ModelAgent.Remark,
		}
	}

	if midjourneyLog.FallbackModelAgent != nil {
		midjourney.IsEnableFallback = true
		midjourney.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     midjourneyLog.FallbackModelAgent.Id,
			ModelAgentName: midjourneyLog.FallbackModelAgent.Name,
		}
	}

	if midjourneyLog.FallbackModel != nil {
		midjourney.IsEnableFallback = true
		if midjourney.FallbackConfig == nil {
			midjourney.FallbackConfig = new(mcommon.FallbackConfig)
		}
		midjourney.FallbackConfig.Model = midjourneyLog.FallbackModel.Model
		midjourney.FallbackConfig.ModelName = midjourneyLog.FallbackModel.Name
	}

	if midjourneyLog.Key != nil {
		midjourney.Key = midjourneyLog.Key.Key
	}

	if midjourneyLog.Response.Response != nil {
		if err := gjson.Unmarshal(midjourneyLog.Response.Response, &midjourney.Response); err != nil {
			logger.Error(ctx, err)
		}
	}

	if midjourneyLog.Response.Error != nil {
		midjourney.ErrMsg = midjourneyLog.Response.Error.Error()
		if common.IsAborted(midjourneyLog.Response.Error) {
			midjourney.Status = 2
		} else {
			midjourney.Status = -1
		}
	}

	if midjourneyLog.RetryInfo != nil {

		midjourney.IsRetry = midjourneyLog.RetryInfo.IsRetry
		midjourney.Retry = &mcommon.Retry{
			IsRetry:    midjourneyLog.RetryInfo.IsRetry,
			RetryCount: midjourneyLog.RetryInfo.RetryCount,
			ErrMsg:     midjourneyLog.RetryInfo.ErrMsg,
		}

		if midjourney.IsRetry {
			midjourney.Status = 3
			midjourney.ErrMsg = midjourneyLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Midjourney.Insert(ctx, midjourney); err != nil {
		logger.Errorf(ctx, "sMidjourney SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			midjourneyLog.Response.Prompt = err.Error()
			midjourneyLog.Response.PromptEn = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sMidjourney SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, midjourneyLog, retry...)
	}
}
