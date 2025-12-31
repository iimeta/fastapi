package embedding

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/v2/consts"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					EmbeddingReq: params,
					Usage:        response.Usage,
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && len(response.Data) > 0 {
					afterHandler.Completion = gconv.String(response.Data[0])
				}

				common.AfterHandler(ctx, mak, afterHandler)

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
