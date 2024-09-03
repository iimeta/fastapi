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
	"math"
	"slices"
	"time"
)

type sEmbedding struct{}

func init() {
	service.RegisterEmbedding(New())
}

func New() service.IEmbedding {
	return &sEmbedding{}
}

// Embeddings
func (s *sEmbedding) Embeddings(ctx context.Context, params sdkm.EmbeddingRequest, fallbackModel *model.Model, retry ...int) (response sdkm.EmbeddingResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sEmbedding Embeddings time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		client      *sdk.EmbeddingClient
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

		if reqModel != nil && response.Usage != nil {
			if reqModel.TextQuota.BillingMethod == 1 {
				totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*reqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*reqModel.TextQuota.CompletionRatio))
			} else {
				totalTokens = reqModel.TextQuota.FixedQuota
			}
		}

		if retryInfo == nil && (err == nil || common.IsAborted(err)) {
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

			if retryInfo == nil && len(response.Data) > 0 && len(response.Data[0].Embedding) > 0 {
				completionsRes.Completion = gconv.String(response.Data[0].Embedding)
			}

			s.SaveLog(ctx, reqModel, realModel, fallbackModel, k, &params, completionsRes, retryInfo)

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
					return s.Embeddings(ctx, params, fallbackModel)
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
						return s.Embeddings(ctx, params, fallbackModel)
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
					return s.Embeddings(ctx, params, fallbackModel)
				}
			}

			return response, err
		}
	}

	request := params
	key = k.Key

	client, err = common.NewEmbeddingClient(ctx, realModel, key, baseUrl, path)
	if err != nil {
		logger.Error(ctx, err)

		if realModel.IsEnableFallback {
			if fallbackModel, _ = service.Model().GetFallbackModel(ctx, realModel); fallbackModel != nil {
				retryInfo = &mcommon.Retry{
					IsRetry:    true,
					RetryCount: len(retry),
					ErrMsg:     err.Error(),
				}
				return s.Embeddings(ctx, params, fallbackModel)
			}
		}

		return response, err
	}

	response, err = client.Embeddings(ctx, request)
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
						return s.Embeddings(ctx, params, fallbackModel)
					}
				}
				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Embeddings(ctx, params, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sEmbedding) SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.EmbeddingRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sEmbedding SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if completionsRes.Error != nil && errors.Is(completionsRes.Error, errors.ERR_MODEL_NOT_FOUND) {
		return
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
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
		chat.Prompt = gconv.String(completionsReq.Input)
	}

	if slices.Contains(config.Cfg.RecordLogs, "completion") {
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

		if chat.IsEnableModelAgent && realModel.ModelAgent != nil {
			chat.ModelAgentId = realModel.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
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

	chat.PromptTokens = completionsRes.Usage.PromptTokens
	chat.CompletionTokens = completionsRes.Usage.CompletionTokens
	chat.TotalTokens = completionsRes.Usage.TotalTokens

	if fallbackModel != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			FallbackModel:     fallbackModel.Model,
			FallbackModelName: fallbackModel.Name,
		}
	}

	if key != nil {
		chat.Key = key.Key
	}

	if completionsRes.Error != nil {
		chat.ErrMsg = completionsRes.Error.Error()
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
		logger.Error(ctx, err)

		if len(retry) == 5 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sEmbedding SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, reqModel, realModel, fallbackModel, key, completionsReq, completionsRes, retryInfo, retry...)
	}
}
