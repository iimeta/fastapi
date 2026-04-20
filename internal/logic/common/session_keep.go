package common

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/internal/config"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func getSessionKeepConfig(modelAgent *mcommon.ModelAgentSessionKeep, enabled bool) *mcommon.ModelAgentSessionKeep {

	var cfg mcommon.ModelAgentSessionKeep

	if config.Cfg.SysConfig.ModelAgentSessionKeep != nil {
		cfg = *config.Cfg.SysConfig.ModelAgentSessionKeep
	}

	if modelAgent != nil {

		if modelAgent.Open {
			cfg.Open = true
		}

		if modelAgent.Ttl > 0 {
			cfg.Ttl = modelAgent.Ttl
		}

		if modelAgent.FailTtl > 0 {
			cfg.FailTtl = modelAgent.FailTtl
		}

		if modelAgent.FailSwitchThreshold > 0 {
			cfg.FailSwitchThreshold = modelAgent.FailSwitchThreshold
		}

		if modelAgent.UserLimit > 0 {
			cfg.UserLimit = modelAgent.UserLimit
		}

		if modelAgent.AgentLimit > 0 {
			cfg.AgentLimit = modelAgent.AgentLimit
		}

		if modelAgent.GlobalLimit > 0 {
			cfg.GlobalLimit = modelAgent.GlobalLimit
		}
	}

	if !enabled {
		cfg.Open = false
	}

	return &cfg
}

func GetResolvedSessionKeepConfig(mak *MAK) *mcommon.ModelAgentSessionKeep {

	if mak == nil || mak.ModelAgent == nil {
		return nil
	}

	return getSessionKeepConfig(mak.ModelAgent.SessionKeepConfig, mak.ModelAgent.IsEnableSessionKeep)
}

func HandleSessionKeepSuccess(ctx context.Context, mak *MAK) {

	if mak == nil || mak.ModelAgent == nil {
		return
	}

	cfg := GetResolvedSessionKeepConfig(mak)
	if cfg == nil || !cfg.Open {
		return
	}

	userId := service.Session().GetUserId(ctx)
	if userId <= 0 {
		userId = service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx())
	}

	if userId <= 0 || mak.RealModel == nil {
		return
	}

	modelName := mak.RealModel.Model
	if modelName == "" {
		modelName = mak.Model
	}

	if modelName == "" {
		return
	}

	if err := service.SessionKeepModelAgent().Refresh(ctx, userId, modelName, mak.ModelAgent.Id); err != nil {
		logger.Error(ctx, err)
	}

	if err := service.SessionKeepModelAgent().ClearFail(ctx, userId, modelName, mak.ModelAgent.Id); err != nil {
		logger.Error(ctx, err)
	}
}

func HandleSessionKeepFailure(ctx context.Context, mak *MAK) {

	if mak == nil || mak.ModelAgent == nil {
		return
	}

	cfg := GetResolvedSessionKeepConfig(mak)
	if cfg == nil || !cfg.Open {
		return
	}

	userId := service.Session().GetUserId(ctx)
	if userId <= 0 {
		userId = service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx())
	}

	if userId <= 0 || mak.RealModel == nil {
		return
	}

	modelName := mak.RealModel.Model
	if modelName == "" {
		modelName = mak.Model
	}

	if modelName == "" {
		return
	}

	failCount, err := service.SessionKeepModelAgent().RecordFail(ctx, userId, modelName, mak.ModelAgent.Id)
	if err != nil {
		logger.Error(ctx, err)
		return
	}

	if cfg.FailSwitchThreshold > 0 && failCount >= cfg.FailSwitchThreshold {
		if err = service.SessionKeepModelAgent().Delete(ctx, userId, modelName, mak.ModelAgent.Id); err != nil {
			logger.Error(ctx, err)
		}
	}
}
