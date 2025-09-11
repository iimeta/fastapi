package common

import (
	"context"
	"fmt"
	"slices"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type MAK struct {
	Provider           string
	Model              string
	Messages           []smodel.ChatCompletionMessage
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
	User               *model.User
	App                *model.App
	AppKey             *model.AppKey
	Group              *model.Group
}

func (mak *MAK) InitMAK(ctx context.Context, retry ...int) (err error) {

	if mak.RealModel == nil {
		mak.RealModel = new(model.Model)
	}

	if mak.Key == nil {
		mak.Key = new(model.Key)
	}

	if mak.User == nil {
		if mak.User, err = service.User().GetCacheUser(ctx, service.Session().GetUserId(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if mak.App == nil {
		if mak.App, err = service.App().GetCacheApp(ctx, service.Session().GetAppId(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if mak.AppKey == nil {
		if mak.AppKey, err = service.AppKey().GetCacheAppKey(ctx, service.Session().GetSecretKey(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if mak.Group == nil {
		if len(mak.User.Groups) > 0 && !mak.AppKey.IsBindGroup && !mak.App.IsBindGroup {

			if mak.ReqModel, mak.Group, err = service.Group().PickGroupModel(ctx, mak.Model, mak.User.Groups...); err != nil {
				logger.Error(ctx, err)
				return err
			}

		} else if mak.AppKey.IsBindGroup {

			if mak.AppKey.Group == "" {
				err = errors.ERR_GROUP_NOT_FOUND
				logger.Error(ctx, err)
				return err
			}

			if mak.Group, err = service.Group().GetCacheGroup(ctx, mak.AppKey.Group); err != nil {
				logger.Error(ctx, err)
				return err
			}

		} else if mak.App.IsBindGroup {

			if mak.App.Group == "" {
				err = errors.ERR_GROUP_NOT_FOUND
				logger.Error(ctx, err)
				return err
			}

			if mak.Group, err = service.Group().GetCacheGroup(ctx, mak.App.Group); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}

	if mak.Group != nil {

		if !slices.Contains(mak.User.Groups, mak.Group.Id) {
			err = errors.ERR_GROUP_NOT_FOUND
			logger.Error(ctx, err)
			return err
		}

		if mak.Group.Status == 2 {
			err = errors.ERR_GROUP_DISABLED
			logger.Error(ctx, err)
			return err
		}

		if mak.Group.ExpiresAt != 0 && mak.Group.ExpiresAt < gtime.TimestampMilli() {
			err = errors.ERR_GROUP_EXPIRED
			logger.Error(ctx, err)
			return err
		}

		if mak.Group.IsLimitQuota && service.Group().GetCacheGroupQuota(ctx, mak.Group.Id) <= 0 {
			err = errors.ERR_GROUP_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}
	}

	if mak.ReqModel == nil && mak.Group != nil {
		if mak.ReqModel, err = service.Model().GetModelByGroup(ctx, mak.Model, mak.Group); err != nil {
			if !mak.Group.IsDefault || !errors.Is(err, errors.ERR_MODEL_NOT_FOUND) {
				logger.Error(ctx, err)
				return err
			}
			mak.Group = nil
		}
	}

	if mak.Group != nil && mak.ReqModel != nil {
		if app, err := service.App().GetCacheApp(ctx, service.Session().GetAppId(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		} else if len(app.Models) > 0 && !slices.Contains(app.Models, mak.ReqModel.Id) {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Info(ctx, err)
			return err
		} else if appKey, err := service.AppKey().GetCacheAppKey(ctx, service.Session().GetSecretKey(ctx)); err != nil {
			logger.Error(ctx, err)
			return err
		} else if len(appKey.Models) > 0 && !slices.Contains(appKey.Models, mak.ReqModel.Id) {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Info(ctx, err)
			return err
		}
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

	if mak.Group != nil && mak.Group.IsEnableForward {
		if mak.RealModel, err = service.Model().GetGroupTargetModel(ctx, mak.Group, mak.RealModel, mak.Messages); err != nil {
			logger.Error(ctx, err)
			return err
		}
	} else if mak.RealModel.IsEnableForward {
		if mak.RealModel, err = service.Model().GetTargetModel(ctx, mak.RealModel, mak.Messages); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	mak.Provider = mak.RealModel.ProviderId
	mak.BaseUrl = mak.RealModel.BaseUrl
	mak.Path = mak.RealModel.Path

	if mak.Group != nil && mak.Group.IsEnableModelAgent {
		if mak.AgentTotal, mak.ModelAgent, err = service.ModelAgent().PickGroupModelAgent(ctx, mak.RealModel, mak.Group); err != nil {
			logger.Error(ctx, err)
			return err
		}
	} else if mak.FallbackModelAgent != nil || mak.RealModel.IsEnableModelAgent {

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
	}

	if mak.ModelAgent != nil {

		mak.Provider = mak.ModelAgent.ProviderId
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

	provider := mak.RealModel.ProviderId

	if mak.ModelAgent != nil {
		provider = mak.ModelAgent.ProviderId
	}

	providerCode := GetProviderCode(ctx, provider)

	if providerCode == sconsts.PROVIDER_GCP_CLAUDE || providerCode == sconsts.PROVIDER_GCP_GEMINI {

		projectId, key, err := getGcpToken(ctx, mak.Key, config.Cfg.Http.ProxyUrl)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}

		mak.RealKey = key

		if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
			for i, replaceModel := range mak.ModelAgent.ReplaceModels {
				if replaceModel == mak.RealModel.Model {
					logger.Infof(ctx, "getRealKey mak.RealModel.Model: %s replaced %s", mak.RealModel.Model, mak.ModelAgent.TargetModels[i])
					mak.RealModel.Model = mak.ModelAgent.TargetModels[i]
					break
				}
			}
		}

		mak.Path = fmt.Sprintf(mak.Path, projectId, mak.RealModel.Model)

	} else if providerCode == sconsts.PROVIDER_BAIDU {
		mak.RealKey = getBaiduToken(ctx, mak.Key.Key, mak.BaseUrl, config.Cfg.Http.ProxyUrl)
	} else {
		mak.RealKey = mak.Key.Key
	}

	return nil
}
