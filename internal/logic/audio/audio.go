package audio

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/api/audio/v1"
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
func (s *sAudio) Speech(ctx context.Context, params sdkm.SpeechRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.SpeechResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio Speech time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              gconv.String(params.Model),
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

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			if mak.ReqModel.AudioQuota.BillingMethod == 1 {
				totalTokens = int(math.Ceil(float64(len(params.Input)) * mak.ReqModel.AudioQuota.PromptRatio))
			} else {
				totalTokens = mak.ReqModel.AudioQuota.FixedQuota
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

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

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, audioReq, audioRes, retryInfo)

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

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.Speech(ctx, request)
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
							return s.Speech(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Speech(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Speech(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// Transcriptions
func (s *sAudio) Transcriptions(ctx context.Context, params *v1.TranscriptionsReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.AudioResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio Transcriptions time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              gconv.String(params.Model),
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		client      sdk.Client
		retryInfo   *mcommon.Retry
		minute      float64
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			if response.Duration != 0 {
				minute = util.Round(response.Duration/60, 2)
			} else {
				minute = util.Round(params.Duration/60, 2)
				response.Duration = params.Duration
			}

			if mak.ReqModel.AudioQuota.BillingMethod == 1 {
				totalTokens = int(math.Ceil(minute * 1000 * mak.ReqModel.AudioQuota.CompletionRatio))
			} else {
				totalTokens = mak.ReqModel.AudioQuota.FixedQuota
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				audioReq := &model.AudioReq{
					FilePath: params.FilePath,
				}

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

				s.SaveLog(ctx, mak.ReqModel, mak.RealModel, mak.ModelAgent, fallbackModelAgent, fallbackModel, mak.Key, audioReq, audioRes, retryInfo)

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

	if client, err = common.NewClient(ctx, mak.Corp, mak.RealModel, mak.RealKey, mak.BaseUrl, mak.Path); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = client.Transcription(ctx, request.AudioRequest)
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
							return s.Transcriptions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Transcriptions(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Transcriptions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sAudio) SaveLog(ctx context.Context, reqModel, realModel *model.Model, modelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, audioReq *model.AudioReq, audioRes *model.AudioRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if audioRes.Error != nil && (errors.Is(audioRes.Error, errors.ERR_MODEL_NOT_FOUND) || errors.Is(audioRes.Error, errors.ERR_MODEL_DISABLED)) {
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
		FilePath:     audioReq.FilePath,
		TotalTokens:  audioRes.TotalTokens,
		TotalTime:    audioRes.TotalTime,
		InternalTime: audioRes.InternalTime,
		ReqTime:      audioRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(audioRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
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
	}

	if audio.IsEnableModelAgent && modelAgent != nil {
		audio.ModelAgentId = modelAgent.Id
		audio.ModelAgent = &do.ModelAgent{
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
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fallbackModelAgent.Id,
			ModelAgentName: fallbackModelAgent.Name,
		}
	}

	if fallbackModel != nil {
		audio.IsEnableFallback = true
		if audio.FallbackConfig == nil {
			audio.FallbackConfig = new(mcommon.FallbackConfig)
		}
		audio.FallbackConfig.Model = fallbackModel.Model
		audio.FallbackConfig.ModelName = fallbackModel.Name
	}

	if key != nil {
		audio.Key = key.Key
	}

	if audioRes.Error != nil {

		audio.ErrMsg = audioRes.Error.Error()
		openaiApiError := &openai.APIError{}
		if errors.As(audioRes.Error, &openaiApiError) {
			audio.ErrMsg = openaiApiError.Message
		}

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
		logger.Errorf(ctx, "sAudio SaveLog error: %v", err)

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sAudio SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, modelAgent, fallbackModelAgent, fallbackModel, key, audioReq, audioRes, retryInfo, retry...)
	}
}
