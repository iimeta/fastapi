package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type MAK struct {
	Corp               string
	Model              string
	Messages           []sdkm.ChatCompletionMessage
	ReqModel           *model.Model
	RealModel          *model.Model
	ModelAgent         *model.ModelAgent
	Key                *model.Key
	FallbackModelAgent *model.ModelAgent
	FallbackModel      *model.Model
	AgentTotal         int
	KeyTotal           int
	RealKey            string
	BaseUrl            string
	Path               string
}

func (mak *MAK) InitMAK(ctx context.Context, retry ...int) (err error) {

	if mak.RealModel == nil {
		mak.RealModel = new(model.Model)
	}

	if mak.ReqModel == nil {
		if mak.ReqModel, err = service.Model().GetModelBySecretKey(ctx, mak.Model, service.Session().GetSecretKey(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if mak.FallbackModel != nil {
		*mak.RealModel = *mak.FallbackModel
	} else {
		*mak.RealModel = *mak.ReqModel
	}

	if mak.RealModel.IsEnableForward {
		if mak.RealModel, err = service.Model().GetTargetModel(ctx, mak.RealModel, mak.Messages); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	mak.Corp = mak.RealModel.Corp
	mak.BaseUrl = mak.RealModel.BaseUrl
	mak.Path = mak.RealModel.Path

	if mak.FallbackModelAgent != nil || mak.RealModel.IsEnableModelAgent {

		if mak.FallbackModelAgent != nil {
			mak.ModelAgent = mak.FallbackModelAgent
			mak.AgentTotal = 1
			mak.RealModel.IsEnableModelAgent = true
		} else {

			if mak.AgentTotal, mak.ModelAgent, err = service.ModelAgent().PickModelAgent(ctx, mak.RealModel); err != nil {
				logger.Error(ctx, err)

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" {
						if mak.FallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); mak.FallbackModelAgent != nil {
							return mak.InitMAK(ctx)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if mak.FallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); mak.FallbackModel != nil {
							return mak.InitMAK(ctx)
						}
					}
				}

				return err
			}
		}

		if mak.ModelAgent != nil {

			mak.Corp = mak.ModelAgent.Corp
			mak.BaseUrl = mak.ModelAgent.BaseUrl
			mak.Path = mak.ModelAgent.Path

			if mak.KeyTotal, mak.Key, err = service.ModelAgent().PickModelAgentKey(ctx, mak.ModelAgent); err != nil {
				logger.Error(ctx, err)

				service.ModelAgent().RecordErrorModelAgent(ctx, mak.RealModel, mak.ModelAgent)

				if errors.Is(err, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
					service.ModelAgent().DisabledModelAgent(ctx, mak.ModelAgent, "No available model agent key")
				}

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if mak.FallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); mak.FallbackModelAgent != nil {
							return mak.InitMAK(ctx)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if mak.FallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); mak.FallbackModel != nil {
							return mak.InitMAK(ctx)
						}
					}
				}

				return err
			}
		}

	} else {

		if mak.KeyTotal, mak.Key, err = service.Key().PickModelKey(ctx, mak.RealModel); err != nil {
			logger.Error(ctx, err)

			if mak.RealModel.IsEnableFallback {

				if mak.RealModel.FallbackConfig.ModelAgent != "" {
					if mak.FallbackModelAgent, _ = service.ModelAgent().GetFallbackModelAgent(ctx, mak.RealModel); mak.FallbackModelAgent != nil {
						return mak.InitMAK(ctx)
					}
				}

				if mak.RealModel.FallbackConfig.Model != "" {
					if mak.FallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); mak.FallbackModel != nil {
						return mak.InitMAK(ctx)
					}
				}
			}

			return err
		}
	}

	if err = getRealKey(ctx, mak); err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := IsNeedRetry(err)

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
			if IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {
				return err
			}
			return mak.InitMAK(ctx, append(retry, 1)...)
		}

		return err
	}

	return nil
}

func getRealKey(ctx context.Context, mak *MAK) error {

	if GetCorpCode(ctx, mak.RealModel.Corp) == consts.CORP_GCP_CLAUDE {

		projectId, key, err := getGcpToken(ctx, mak.Key, config.Cfg.Http.ProxyUrl)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}

		mak.RealKey = key
		mak.Path = fmt.Sprintf(mak.Path, projectId, mak.RealModel.Model)

	} else if GetCorpCode(ctx, mak.RealModel.Corp) == consts.CORP_BAIDU {
		mak.RealKey = getBaiduToken(ctx, mak.Key.Key, mak.BaseUrl, config.Cfg.Http.ProxyUrl)
	} else {
		mak.RealKey = mak.Key.Key
	}

	return nil
}
