package embedding

import (
	"context"
	"slices"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	serrors "github.com/iimeta/fastapi-sdk/errors"
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

type sEmbedding struct{}

func init() {
	service.RegisterEmbedding(New())
}

func New() service.IEmbedding {
	return &sEmbedding{}
}

// Embeddings
func (s *sEmbedding) Embeddings(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.EmbeddingResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sEmbedding Embeddings time: %d", gtime.TimestampMilli()-now)
	}()

	params, err := common.NewConverter(ctx, sconsts.PROVIDER_OPENAI).ConvTextEmbeddingsRequest(ctx, data)
	if err != nil {
		logger.Errorf(ctx, "sEmbedding Embeddings ConvTextEmbeddingsRequest error: %v", err)
		return response, err
	}

	if params.Input == nil || len(gconv.SliceAny(params.Input)) == 0 {
		return response, errors.ERR_INVALID_PARAMETER
	}

	var (
		mak = &common.MAK{
			Model:              gconv.String(params.Model),
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		spend     mcommon.Spend
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			billingData := &mcommon.BillingData{
				EmbeddingRequest: params,
				Usage:            response.Usage,
			}

			if len(response.Data) > 0 {
				billingData.Completion = gconv.String(response.Data[0])
			}

			// 花费
			spend = common.Spend(ctx, mak, billingData)
			response.Usage = billingData.Usage

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, spend.TotalTokens, mak.Key.Key, mak.Group); err != nil {
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
					completionsRes.Usage.TotalTokens = spend.TotalTokens
				}

				if retryInfo == nil && len(response.Data) > 0 {
					completionsRes.Completion = gconv.String(response.Data[0])
				}

				s.SaveLog(ctx, model.ChatLog{
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					EmbeddingReq:       &params,
					CompletionsRes:     completionsRes,
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

	request := params

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sEmbedding Embeddings request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				break
			}
		}
	}

	response, err = common.NewAdapter(ctx, mak, false).TextEmbeddings(ctx, data)
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
							return s.Embeddings(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Embeddings(g.RequestFromCtx(ctx).GetCtx(), data, nil, fallbackModel)
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

			return s.Embeddings(g.RequestFromCtx(ctx).GetCtx(), data, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// 保存日志
func (s *sEmbedding) SaveLog(ctx context.Context, chatLog model.ChatLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sEmbedding SaveLog time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if chatLog.CompletionsRes.Error != nil && (errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	chat := do.Chat{
		TraceId:          gctx.CtxId(ctx),
		UserId:           service.Session().GetUserId(ctx),
		AppId:            service.Session().GetAppId(ctx),
		PromptTokens:     chatLog.CompletionsRes.Usage.PromptTokens,
		CompletionTokens: chatLog.CompletionsRes.Usage.CompletionTokens,
		Spend:            chatLog.Spend,
		ConnTime:         chatLog.CompletionsRes.ConnTime,
		Duration:         chatLog.CompletionsRes.Duration,
		TotalTime:        chatLog.CompletionsRes.TotalTime,
		InternalTime:     chatLog.CompletionsRes.InternalTime,
		ReqTime:          chatLog.CompletionsRes.EnterTime,
		ReqDate:          gtime.NewFromTimeStamp(chatLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:         g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:         g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:          util.GetLocalIp(),
		Status:           1,
		Host:             g.RequestFromCtx(ctx).GetHost(),
		Rid:              service.Session().GetRid(ctx),
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {
		chat.Prompt = gconv.String(chatLog.EmbeddingReq.Input)
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") {
		chat.Completion = chatLog.CompletionsRes.Completion
	}

	if chatLog.ReqModel != nil {
		chat.ProviderId = chatLog.ReqModel.ProviderId
		chat.ModelId = chatLog.ReqModel.Id
		chat.ModelName = chatLog.ReqModel.Name
		chat.Model = chatLog.ReqModel.Model
		chat.ModelType = chatLog.ReqModel.Type
	}

	if chatLog.RealModel != nil {
		chat.IsEnablePresetConfig = chatLog.RealModel.IsEnablePresetConfig
		chat.PresetConfig = chatLog.RealModel.PresetConfig
		chat.IsEnableForward = chatLog.RealModel.IsEnableForward
		chat.ForwardConfig = chatLog.RealModel.ForwardConfig
		chat.IsEnableModelAgent = chatLog.RealModel.IsEnableModelAgent
		chat.RealModelId = chatLog.RealModel.Id
		chat.RealModelName = chatLog.RealModel.Name
		chat.RealModel = chatLog.RealModel.Model
	}

	if chatLog.ModelAgent != nil {
		chat.IsEnableModelAgent = true
		chat.ModelAgentId = chatLog.ModelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			ProviderId: chatLog.ModelAgent.ProviderId,
			Name:       chatLog.ModelAgent.Name,
			BaseUrl:    chatLog.ModelAgent.BaseUrl,
			Path:       chatLog.ModelAgent.Path,
			Weight:     chatLog.ModelAgent.Weight,
			Remark:     chatLog.ModelAgent.Remark,
		}
	}

	if chatLog.FallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     chatLog.FallbackModelAgent.Id,
			ModelAgentName: chatLog.FallbackModelAgent.Name,
		}
	}

	if chatLog.FallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = chatLog.FallbackModel.Model
		chat.FallbackConfig.ModelName = chatLog.FallbackModel.Name
	}

	if chatLog.Key != nil {
		chat.Key = chatLog.Key.Key
	}

	if chatLog.CompletionsRes.Error != nil {

		chat.ErrMsg = chatLog.CompletionsRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(chatLog.CompletionsRes.Error, &openaiApiError) {
			chat.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(chatLog.CompletionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if chatLog.RetryInfo != nil {

		chat.IsRetry = chatLog.RetryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    chatLog.RetryInfo.IsRetry,
			RetryCount: chatLog.RetryInfo.RetryCount,
			ErrMsg:     chatLog.RetryInfo.ErrMsg,
		}

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = chatLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sEmbedding SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			chatLog.EmbeddingReq.Input = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sEmbedding SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, chatLog, retry...)
	}
}
