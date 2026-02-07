package midjourney

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk/v2"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					MidjourneyPath:     path,
					MidjourneyReqUrl:   reqUrl,
					MidjourneyTaskId:   taskId,
					MidjourneyPrompt:   prompt,
					MidjourneyResponse: response,
					Error:              err,
					RetryInfo:          retryInfo,
					TotalTime:          response.TotalTime,
					InternalTime:       internalTime,
					EnterTime:          enterTime,
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

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "midjourney")
		if spend.Midjourney == nil || spend.Midjourney.Pricing == nil || spend.Midjourney.Pricing.Path == "" {
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

				if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
					service.Session().SaveModelAgentBillingMethod(ctx, 1)
					retry = []int{}
				} else {

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

	data := map[string]any{}
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
	)

	if model := request.GetRouterMap()["model"]; model != "" {
		defaultModel = model
		mak.Model = model
		path = gstr.Replace(path, "/"+defaultModel, "")
	}

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					MidjourneyPath:   path,
					MidjourneyTaskId: taskId,
					MidjourneyResponse: smodel.MidjourneyResponse{
						Response: []byte(fmt.Sprintf("taskId: %s\nimageUrl: %s", taskId, imageUrl)),
					},
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
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

		// 计算花费
		spend := common.Billing(ctx, mak, billingData, "midjourney")
		if spend.Midjourney == nil || spend.Midjourney.Pricing == nil || spend.Midjourney.Pricing.Path == "" {
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

				if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
					service.Session().SaveModelAgentBillingMethod(ctx, 1)
					retry = []int{}
				} else {

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

	data := map[string]any{}
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
