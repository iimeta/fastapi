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
	v1 "github.com/iimeta/fastapi/api/audio/v1"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"math"
	"time"
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

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {

			if reqModel != nil {
				if reqModel.AudioQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(float64(len(params.Input)) * reqModel.AudioQuota.PromptRatio))
				} else {
					totalTokens = reqModel.AudioQuota.FixedQuota
				}
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

			realModel.ModelAgent = modelAgent

			audioReq := &model.AudioReq{
				Input: params.Input,
			}

			audioRes := &model.AudioRes{
				Characters:   len(audioReq.Input),
				Error:        err,
				TotalTime:    response.TotalTime,
				InternalTime: internalTime,
				EnterTime:    enterTime,
			}

			if retryInfo == nil && (err == nil || common.IsAborted(err)) {
				audioRes.TotalTokens = totalTokens
			}

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, audioReq, audioRes, retryInfo)

		}); err != nil {
			logger.Error(ctx, err)
		}
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
func (s *sAudio) Transcriptions(ctx context.Context, params *v1.TranscriptionsReq, fallbackModel *model.Model, retry ...int) (response sdkm.AudioResponse, err error) {

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
		minute      float64
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {

			if response.Duration != 0 {
				minute = util.Round(response.Duration/60, 2)
			} else {
				minute = util.Round(params.Duration/60, 2)
			}

			if reqModel != nil {
				if reqModel.AudioQuota.BillingMethod == 1 {
					totalTokens = int(math.Ceil(minute * 1000 * reqModel.AudioQuota.CompletionRatio))
				} else {
					totalTokens = reqModel.AudioQuota.FixedQuota
				}
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, k.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

			realModel.ModelAgent = modelAgent

			audioReq := &model.AudioReq{}

			audioRes := &model.AudioRes{
				Text:         response.Text,
				Minute:       minute,
				Error:        err,
				TotalTime:    response.TotalTime,
				InternalTime: internalTime,
				EnterTime:    enterTime,
			}

			if retryInfo == nil {
				audioRes.TotalTokens = totalTokens
			}

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, audioReq, audioRes, retryInfo)

		}); err != nil {
			logger.Error(ctx, err)
		}
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

	response, err = client.Transcription(ctx, request.AudioRequest)
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
func (s *sAudio) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, audioReq *model.AudioReq, audioRes *model.AudioRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if audioRes.Error != nil && errors.Is(audioRes.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	audio := do.Audio{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Input:        audioReq.Input,
		Text:         audioRes.Text,
		Characters:   audioRes.Characters,
		Minute:       audioRes.Minute,
		TotalTokens:  audioRes.TotalTokens,
		TotalTime:    audioRes.TotalTime,
		InternalTime: audioRes.InternalTime,
		ReqTime:      audioRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(audioRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
	}

	if reqModel != nil {
		audio.Corp = reqModel.Corp
		audio.ModelId = reqModel.Id
		audio.Name = reqModel.Name
		audio.Model = reqModel.Model
		audio.Type = reqModel.Type
		audio.AudioQuota = reqModel.AudioQuota
	}

	if realModel != nil {

		audio.IsEnablePresetConfig = realModel.IsEnablePresetConfig
		audio.PresetConfig = realModel.PresetConfig
		audio.IsEnableForward = realModel.IsEnableForward
		audio.ForwardConfig = realModel.ForwardConfig
		audio.IsEnableModelAgent = realModel.IsEnableModelAgent
		audio.RealModelId = realModel.Id
		audio.RealModelName = realModel.Name
		audio.RealModel = realModel.Model

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

	}

	if fallbackModel != nil {
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

	if key != nil {
		audio.Key = key.Key
	}

	if audioRes.Error != nil {
		audio.ErrMsg = audioRes.Error.Error()
		if common.IsAborted(audioRes.Error) {
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

		if audio.IsRetry {
			audio.Status = 3
			audio.ErrMsg = retryInfo.ErrMsg
		}
	}

	if _, err := dao.Audio.Insert(ctx, audio); err != nil {
		logger.Error(ctx, err)

		if len(retry) == 5 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sAudio SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModel, key, audioReq, audioRes, retryInfo, retry...)
	}
}
