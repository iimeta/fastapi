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
	"slices"
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

				s.SaveLog(ctx, model.AudioLog{
					Group:              mak.Group,
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					AudioReq:           audioReq,
					AudioRes:           audioRes,
					RetryInfo:          retryInfo,
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

	request := params

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if openai.SpeechModel(replaceModel) == request.Model {
				logger.Infof(ctx, "sAudio Speech request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = openai.SpeechModel(mak.ModelAgent.TargetModels[i])
				break
			}
		}
	}

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

				s.SaveLog(ctx, model.AudioLog{
					Group:              mak.Group,
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					AudioReq:           audioReq,
					AudioRes:           audioRes,
					RetryInfo:          retryInfo,
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

	request := params

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sAudio Transcriptions request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
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
func (s *sAudio) SaveLog(ctx context.Context, audioLog model.AudioLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAudio SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if audioLog.AudioRes.Error != nil && (errors.Is(audioLog.AudioRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(audioLog.AudioRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(audioLog.AudioRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(audioLog.AudioRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(audioLog.AudioRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(audioLog.AudioRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	audio := do.Audio{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Input:        audioLog.AudioReq.Input,
		Text:         audioLog.AudioRes.Text,
		Characters:   audioLog.AudioRes.Characters,
		Minute:       audioLog.AudioRes.Minute,
		FilePath:     audioLog.AudioReq.FilePath,
		TotalTokens:  audioLog.AudioRes.TotalTokens,
		TotalTime:    audioLog.AudioRes.TotalTime,
		InternalTime: audioLog.AudioRes.InternalTime,
		ReqTime:      audioLog.AudioRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(audioLog.AudioRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if audioLog.Group != nil {
		audio.GroupId = audioLog.Group.Id
		audio.GroupName = audioLog.Group.Name
		audio.Discount = audioLog.Group.Discount
	}

	if audioLog.ReqModel != nil {
		audio.Corp = audioLog.ReqModel.Corp
		audio.ModelId = audioLog.ReqModel.Id
		audio.Name = audioLog.ReqModel.Name
		audio.Model = audioLog.ReqModel.Model
		audio.Type = audioLog.ReqModel.Type
		audio.AudioQuota = audioLog.ReqModel.AudioQuota
	}

	if audioLog.RealModel != nil {
		audio.IsEnablePresetConfig = audioLog.RealModel.IsEnablePresetConfig
		audio.PresetConfig = audioLog.RealModel.PresetConfig
		audio.IsEnableForward = audioLog.RealModel.IsEnableForward
		audio.ForwardConfig = audioLog.RealModel.ForwardConfig
		audio.IsEnableModelAgent = audioLog.RealModel.IsEnableModelAgent
		audio.RealModelId = audioLog.RealModel.Id
		audio.RealModelName = audioLog.RealModel.Name
		audio.RealModel = audioLog.RealModel.Model
	}

	if audio.IsEnableModelAgent && audioLog.ModelAgent != nil {
		audio.ModelAgentId = audioLog.ModelAgent.Id
		audio.ModelAgent = &do.ModelAgent{
			Corp:    audioLog.ModelAgent.Corp,
			Name:    audioLog.ModelAgent.Name,
			BaseUrl: audioLog.ModelAgent.BaseUrl,
			Path:    audioLog.ModelAgent.Path,
			Weight:  audioLog.ModelAgent.Weight,
			Remark:  audioLog.ModelAgent.Remark,
			Status:  audioLog.ModelAgent.Status,
		}
	}

	if audioLog.FallbackModelAgent != nil {
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     audioLog.FallbackModelAgent.Id,
			ModelAgentName: audioLog.FallbackModelAgent.Name,
		}
	}

	if audioLog.FallbackModel != nil {
		audio.IsEnableFallback = true
		if audio.FallbackConfig == nil {
			audio.FallbackConfig = new(mcommon.FallbackConfig)
		}
		audio.FallbackConfig.Model = audioLog.FallbackModel.Model
		audio.FallbackConfig.ModelName = audioLog.FallbackModel.Name
	}

	if audioLog.Key != nil {
		audio.Key = audioLog.Key.Key
	}

	if audioLog.AudioRes.Error != nil {

		audio.ErrMsg = audioLog.AudioRes.Error.Error()
		openaiApiError := &openai.APIError{}
		if errors.As(audioLog.AudioRes.Error, &openaiApiError) {
			audio.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(audioLog.AudioRes.Error) {
			audio.Status = 2
		} else {
			audio.Status = -1
		}
	}

	if audioLog.RetryInfo != nil {

		audio.IsRetry = audioLog.RetryInfo.IsRetry
		audio.Retry = &mcommon.Retry{
			IsRetry:    audioLog.RetryInfo.IsRetry,
			RetryCount: audioLog.RetryInfo.RetryCount,
			ErrMsg:     audioLog.RetryInfo.ErrMsg,
		}

		if audio.IsRetry {
			audio.Status = 3
			audio.ErrMsg = audioLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Audio.Insert(ctx, audio); err != nil {
		logger.Errorf(ctx, "sAudio SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			audioLog.AudioReq.Input = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sAudio SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, audioLog, retry...)
	}
}
