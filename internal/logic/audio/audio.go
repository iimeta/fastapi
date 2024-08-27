package chat

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sdk "github.com/iimeta/fastapi-sdk"
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
	"slices"
)

type sAudio struct{}

func init() {
	service.RegisterAudio(New())
}

func New() service.IAudio {
	return &sAudio{}
}

// Speech
func (s *sAudio) Speech(ctx context.Context, params sdkm.SpeechRequest, fallbackModel *model.Model, retry ...int) (response sdkm.SpeechResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio Speech time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		client      sdk.Client
		reqModel    *model.Model
		realModel   = new(model.Model)
		k           *model.Key
		modelAgent  *model.ModelAgent
		key         string
		baseUrl     string
		path        string
		agentTotal  int
		keyTotal    int
		retryInfo   *mcommon.Retry
		totalTokens int
	)

	defer func() {

		//enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		//internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		//if reqModel != nil && response.Usage != nil {
		//	// 实际花费额度
		//	if reqModel.TextQuota.BillingMethod == 1 {
		//		totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
		//	} else {
		//		totalTokens = reqModel.TextQuota.FixedQuota
		//	}
		//}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		//if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		//
		//	realModel.ModelAgent = modelAgent
		//
		//	completionsRes := &model.CompletionsRes{
		//		Error:        err,
		//		TotalTime:    response.TotalTime,
		//		InternalTime: internalTime,
		//		EnterTime:    enterTime,
		//	}
		//
		//	if retryInfo == nil && response.Usage != nil {
		//		completionsRes.Usage = *response.Usage
		//		completionsRes.Usage.TotalTokens = totalTokens
		//	}
		//
		//	if retryInfo == nil && len(response.Data) > 0 && len(response.Data[0].Audio) > 0 {
		//		completionsRes.Completion = gconv.String(response.Data[0].Audio)
		//	}
		//
		//	s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo)
		//
		//}, nil); err != nil {
		//	logger.Error(ctx, err)
		//}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, gconv.String(params.Model), service.Session().GetSecretKey(ctx)); err != nil {
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
					return s.Speech(ctx, params, fallbackModel)
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
						return s.Speech(ctx, params, fallbackModel)
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
					return s.Speech(ctx, params, fallbackModel)
				}
			}

			return response, err
		}
	}

	request := params
	key = k.Key

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
				return s.Speech(ctx, params, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.Speech(ctx, request)
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
						return s.Speech(ctx, params, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Speech(ctx, params, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// Transcriptions
func (s *sAudio) Transcriptions(ctx context.Context, params sdkm.AudioRequest, fallbackModel *model.Model, retry ...int) (response sdkm.AudioResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio Transcriptions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		client      sdk.Client
		reqModel    *model.Model
		realModel   = new(model.Model)
		k           *model.Key
		modelAgent  *model.ModelAgent
		key         string
		baseUrl     string
		path        string
		agentTotal  int
		keyTotal    int
		retryInfo   *mcommon.Retry
		totalTokens int
	)

	defer func() {

		//enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		//internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		//if reqModel != nil && response.Usage != nil {
		//	// 实际花费额度
		//	if reqModel.TextQuota.BillingMethod == 1 {
		//		totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
		//	} else {
		//		totalTokens = reqModel.TextQuota.FixedQuota
		//	}
		//}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		//if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		//
		//	realModel.ModelAgent = modelAgent
		//
		//	completionsRes := &model.CompletionsRes{
		//		Error:        err,
		//		TotalTime:    response.TotalTime,
		//		InternalTime: internalTime,
		//		EnterTime:    enterTime,
		//	}
		//
		//	if retryInfo == nil && response.Usage != nil {
		//		completionsRes.Usage = *response.Usage
		//		completionsRes.Usage.TotalTokens = totalTokens
		//	}
		//
		//	if retryInfo == nil && len(response.Data) > 0 && len(response.Data[0].Audio) > 0 {
		//		completionsRes.Completion = gconv.String(response.Data[0].Audio)
		//	}
		//
		//	s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo)
		//
		//}, nil); err != nil {
		//	logger.Error(ctx, err)
		//}
	}()

	if reqModel, err = service.Model().GetModelBySecretKey(ctx, gconv.String(params.Model), service.Session().GetSecretKey(ctx)); err != nil {
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
					return s.Transcriptions(ctx, params, fallbackModel)
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
						return s.Transcriptions(ctx, params, fallbackModel)
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
					return s.Transcriptions(ctx, params, fallbackModel)
				}
			}

			return response, err
		}
	}

	request := params
	key = k.Key

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
				return s.Transcriptions(ctx, params, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.Transcription(ctx, request)
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
						return s.Transcriptions(ctx, params, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Transcriptions(ctx, params, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sAudio) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.AudioRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch ...bool) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	audio := do.Audio{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: len(isSmartMatch) > 0 && isSmartMatch[0],
		ConnTime:     completionsRes.ConnTime,
		Duration:     completionsRes.Duration,
		TotalTime:    completionsRes.TotalTime,
		InternalTime: completionsRes.InternalTime,
		ReqTime:      completionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(completionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
	}

	if slices.Contains(config.Cfg.RecordLogs, "prompt") {
		//audio.Prompt = gconv.String(completionsReq.Input)
	}

	if slices.Contains(config.Cfg.RecordLogs, "completion") {
		//audio.Completion = completionsRes.Completion
	}

	audio.Corp = reqModel.Corp
	audio.ModelId = reqModel.Id
	audio.Name = reqModel.Name
	audio.Model = reqModel.Model
	audio.Type = reqModel.Type
	//audio.TextQuota = reqModel.TextQuota
	//audio.MultimodalQuota = reqModel.MultimodalQuota

	audio.IsEnablePresetConfig = realModel.IsEnablePresetConfig
	audio.PresetConfig = realModel.PresetConfig
	audio.IsEnableForward = realModel.IsEnableForward
	audio.ForwardConfig = realModel.ForwardConfig
	audio.IsEnableModelAgent = realModel.IsEnableModelAgent
	audio.RealModelId = realModel.Id
	audio.RealModelName = realModel.Name
	audio.RealModel = realModel.Model

	//audio.PromptTokens = completionsRes.Usage.PromptTokens
	//audio.CompletionTokens = completionsRes.Usage.CompletionTokens
	audio.TotalTokens = completionsRes.Usage.TotalTokens

	if fallbackModel != nil {
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

	if audio.IsEnableModelAgent && realModel.ModelAgent != nil {
		audio.ModelAgentId = realModel.ModelAgent.Id
		audio.ModelAgent = &do.ModelAgent{
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
		audio.Key = key.Key
	}

	if completionsRes.Error != nil {
		audio.ErrMsg = completionsRes.Error.Error()
		if common.IsAborted(completionsRes.Error) {
			audio.Status = 2
		} else {
			audio.Status = -1
		}
	}

	if retryInfo != nil {

		audio.IsRetry = retryInfo.IsRetry
		audio.Retry = &mcommon.Retry{
			IsRetry:    retryInfo.IsRetry,
			RetryCount: retryInfo.RetryCount,
			ErrMsg:     retryInfo.ErrMsg,
		}

		if audio.IsRetry && completionsRes.Error == nil {
			audio.Status = 3
			audio.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, audio); err != nil {
		logger.Error(ctx, err)
	}
}
