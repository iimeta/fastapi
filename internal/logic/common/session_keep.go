package common

import (
	"context"

	"github.com/gogf/gf/v2/os/gtime"
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

		if modelAgent.Mode != "" {
			cfg.Mode = modelAgent.Mode
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

		if len(modelAgent.Rules) > 0 {
			cfg.Rules = modelAgent.Rules
		}

		if modelAgent.EnableSystemPromptHash {
			cfg.EnableSystemPromptHash = true
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

// 会话保持成功处理
func HandleSessionKeepSuccess(ctx context.Context, mak *MAK) {

	if mak == nil || mak.ModelAgent == nil || !mak.ModelAgent.IsEnableSessionKeep {
		return
	}

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "HandleSessionKeepSuccess time: %d", gtime.TimestampMilli()-now)
	}()

	cfg := GetResolvedSessionKeepConfig(mak)
	if cfg == nil || !cfg.Open {
		return
	}

	sk := service.Session().GetSessionKey(ctx)
	if sk == nil {
		modelName := ""
		if mak.RealModel != nil {
			modelName = mak.RealModel.Name
		}
		if modelName == "" {
			modelName = mak.Model
		}
		if modelName == "" {
			return
		}
		sk = service.SessionKeepModelAgent().ResolveSessionKey(ctx, modelName, cfg)
	}
	if sk == nil {
		return
	}

	keyId := ""
	if mak.Key != nil {
		keyId = mak.Key.Id
	}

	if err := service.SessionKeepModelAgent().Refresh(ctx, sk, mak.ModelAgent.Id, keyId); err != nil {
		logger.Error(ctx, err)
	}

	if err := service.SessionKeepModelAgent().ClearFail(ctx, sk, mak.ModelAgent.Id); err != nil {
		logger.Error(ctx, err)
	}

	if keyId != "" {
		if err := service.SessionKeepModelAgent().ClearKeyFail(ctx, sk, mak.ModelAgent.Id, keyId); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 会话保持失败处理
func HandleSessionKeepFailure(ctx context.Context, mak *MAK) {

	if mak == nil || mak.ModelAgent == nil || !mak.ModelAgent.IsEnableSessionKeep {
		return
	}

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "HandleSessionKeepFailure time: %d", gtime.TimestampMilli()-now)
	}()

	cfg := GetResolvedSessionKeepConfig(mak)
	if cfg == nil || !cfg.Open {
		return
	}

	sk := service.Session().GetSessionKey(ctx)
	if sk == nil {
		modelName := ""
		if mak.RealModel != nil {
			modelName = mak.RealModel.Name
		}
		if modelName == "" {
			modelName = mak.Model
		}
		if modelName == "" {
			return
		}
		sk = service.SessionKeepModelAgent().ResolveSessionKey(ctx, modelName, cfg)
	}
	if sk == nil {
		return
	}

	keyId := ""
	if mak.Key != nil {
		keyId = mak.Key.Id
	}

	if keyId != "" {
		keyFailCount, keyErr := service.SessionKeepModelAgent().RecordKeyFail(ctx, sk, mak.ModelAgent.Id, keyId)
		if keyErr != nil {
			logger.Error(ctx, keyErr)
			return
		}

		if cfg.FailSwitchThreshold > 0 && keyFailCount >= cfg.FailSwitchThreshold {
			if err := service.SessionKeepModelAgent().Refresh(ctx, sk, mak.ModelAgent.Id, ""); err != nil {
				logger.Error(ctx, err)
			}
			return
		}
	}

	failCount, err := service.SessionKeepModelAgent().RecordFail(ctx, sk, mak.ModelAgent.Id)
	if err != nil {
		logger.Error(ctx, err)
		return
	}

	if cfg.FailSwitchThreshold > 0 && failCount >= cfg.FailSwitchThreshold {
		if err = service.SessionKeepModelAgent().Delete(ctx, sk, mak.ModelAgent.Id); err != nil {
			logger.Error(ctx, err)
		}
	}
}
