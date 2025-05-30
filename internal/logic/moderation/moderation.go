package moderation

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
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
	"github.com/iimeta/go-openai"
	"math"
	"slices"
	"time"
)

type sModeration struct{}

func init() {
	service.RegisterModeration(New())
}

func New() service.IModeration {
	return &sModeration{}
}

// Moderations
func (s *sModeration) Moderations(ctx context.Context, params sdkm.ModerationRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ModerationResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModeration Moderations time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      sdk.Client
		retryInfo   *mcommon.Retry
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && response.Usage != nil {
			if mak.ReqModel.TextQuota.BillingMethod == 1 {
				totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))
			} else {
				totalTokens = mak.ReqModel.TextQuota.FixedQuota
			}
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			// 分组折扣
			if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
				totalTokens = int(math.Ceil(float64(totalTokens) * mak.Group.Discount))
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key, mak.Group); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				completionsRes := &model.CompletionsRes{
					Error:        err,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && response.Usage != nil {
					completionsRes.Usage = *response.Usage
					completionsRes.Usage.TotalTokens = totalTokens
				}

				if retryInfo == nil && response.Results != nil {
					completionsRes.Completion = gconv.String(response.Results)
				}

				s.SaveLog(ctx, mak.Group, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, &params, completionsRes, retryInfo)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	request := params

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sModeration Moderations request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.Moderations(ctx, request)
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
							return s.Moderations(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Moderations(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Moderations(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sModeration) SaveLog(ctx context.Context, group *model.Group, reqModel, realModel *model.Model, modelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ModerationRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModeration SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && (errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(completionsRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(completionsRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(completionsRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(completionsRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(completionsRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	chat := do.Chat{
		TraceId:          gctx.CtxId(ctx),
		UserId:           service.Session().GetUserId(ctx),
		AppId:            service.Session().GetAppId(ctx),
		PromptTokens:     completionsRes.Usage.PromptTokens,
		CompletionTokens: completionsRes.Usage.CompletionTokens,
		TotalTokens:      completionsRes.Usage.TotalTokens,
		ConnTime:         completionsRes.ConnTime,
		Duration:         completionsRes.Duration,
		TotalTime:        completionsRes.TotalTime,
		InternalTime:     completionsRes.InternalTime,
		ReqTime:          completionsRes.EnterTime,
		ReqDate:          gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:         g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:         g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:          util.GetLocalIp(),
		Status:           1,
		Host:             g.RequestFromCtx(ctx).GetHost(),
		Rid:              service.Session().GetRid(ctx),
	}

	if group != nil {
		chat.GroupId = group.Id
		chat.GroupName = group.Name
		chat.Discount = group.Discount
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {
		chat.Prompt = gconv.String(completionsReq.Input)
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") {
		chat.Completion = completionsRes.Completion
	}

	if reqModel != nil {
		chat.Corp = reqModel.Corp
		chat.ModelId = reqModel.Id
		chat.Name = reqModel.Name
		chat.Model = reqModel.Model
		chat.Type = reqModel.Type
		chat.TextQuota = reqModel.TextQuota
		chat.MultimodalQuota = reqModel.MultimodalQuota
	}

	if realModel != nil {
		chat.IsEnablePresetConfig = realModel.IsEnablePresetConfig
		chat.PresetConfig = realModel.PresetConfig
		chat.IsEnableForward = realModel.IsEnableForward
		chat.ForwardConfig = realModel.ForwardConfig
		chat.IsEnableModelAgent = realModel.IsEnableModelAgent
		chat.RealModelId = realModel.Id
		chat.RealModelName = realModel.Name
		chat.RealModel = realModel.Model
	}

	if chat.IsEnableModelAgent && modelAgent != nil {
		chat.ModelAgentId = modelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			Corp:    modelAgent.Corp,
			Name:    modelAgent.Name,
			BaseUrl: modelAgent.BaseUrl,
			Path:    modelAgent.Path,
			Weight:  modelAgent.Weight,
			Remark:  modelAgent.Remark,
			Status:  modelAgent.Status,
		}
	}

	if fallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = fallbackModel.Model
		chat.FallbackConfig.ModelName = fallbackModel.Name
	}

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Error != nil {

		chat.ErrMsg = completionsRes.Error.Error()
		openaiApiError := &openai.APIError{}
		if errors.As(completionsRes.Error, &openaiApiError) {
			chat.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(completionsRes.Error) {
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

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sModeration SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			completionsReq.Input = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sModeration SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, group, reqModel, realModel, modelAgent, fallbackModelAgent, fallbackModel, key, completionsReq, completionsRes, retryInfo, retry...)
	}
}
