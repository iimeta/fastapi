package model_agent_session_keep

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
)

const (
	defaultSessionKeepTTL         = 30 * time.Minute
	defaultSessionKeepFailureTTL  = 5 * time.Minute
	defaultSessionKeepSwitchLimit = int64(3)
	defaultSessionKeepUserLimit   = int64(50)
	defaultSessionKeepAgentLimit  = int64(10000)
	defaultSessionKeepGlobalLimit = int64(200000)

	sessionKeepValuePrefix    = "session:agent:u:%d:m:%s"
	sessionKeepFailPrefix     = "session:agent:fail:u:%d:m:%s:a:%s"
	sessionKeepAgentSetPrefix = "session:agent:set:%s"
	sessionKeepUserAgentSet   = "session:agent:user:set:%d:a:%s"
	sessionKeepGlobalSet      = "session:agent:global"
	sessionKeepLocalKeyPrefix = "session_keep:%d:%s"
)

type localSessionKeepValue struct {
	AgentId string
}

type sModelAgentSessionKeep struct {
	localCache *cache.Cache
}

func init() {
	service.RegisterModelAgentSessionKeep(New())
}

func New() service.IModelAgentSessionKeep {
	return &sModelAgentSessionKeep{
		localCache: cache.New(),
	}
}

func (s *sModelAgentSessionKeep) Get(ctx context.Context, userId int, modelName string) (string, bool, error) {

	key := s.localKey(userId, modelName)
	if value := s.localCache.GetVal(ctx, key); value != nil {
		if localValue, ok := value.(*localSessionKeepValue); ok && localValue.AgentId != "" {
			return localValue.AgentId, true, nil
		}
	}

	agentId, err := redis.GetStr(ctx, s.redisValueKey(userId, modelName))
	if err != nil {
		return "", false, err
	}

	if agentId == "" {
		return "", false, nil
	}

	cfg, cfgErr := s.cfgByAgent(ctx, agentId)
	if cfgErr != nil {
		return "", false, cfgErr
	}

	if err = s.localCache.Set(ctx, key, &localSessionKeepValue{AgentId: agentId}, cfg.Ttl); err != nil {
		logger.Error(ctx, err)
	}

	return agentId, true, nil
}

func (s *sModelAgentSessionKeep) Set(ctx context.Context, userId int, modelName, agentId string) error {

	if agentId == "" || modelName == "" || userId <= 0 {
		return nil
	}

	if err := s.cleanupExpired(ctx, userId, modelName); err != nil {
		return err
	}

	cfg, err := s.cfgByAgent(ctx, agentId)
	if err != nil {
		return err
	}

	oldAgentId, exists, err := s.getStoredAgentId(ctx, userId, modelName)
	if err != nil {
		return err
	}

	if exists && oldAgentId != "" && oldAgentId != agentId {
		if err = s.removeIndex(ctx, userId, modelName, oldAgentId); err != nil {
			return err
		}
	}

	if err = s.ensureLimit(ctx, userId, modelName, agentId, cfg); err != nil {
		return err
	}

	if err = redis.SetEX(ctx, s.redisValueKey(userId, modelName), agentId, int64(cfg.Ttl)); err != nil {
		return err
	}

	now := time.Now().Unix()
	member := s.member(userId, modelName)
	if _, err := redis.ZAdd(ctx, s.redisAgentSetKey(agentId), float64(now), member); err != nil {
		return err
	}

	if _, err := redis.ZAdd(ctx, s.redisUserAgentSetKey(userId, agentId), float64(now), modelName); err != nil {
		return err
	}

	if _, err := redis.ZAdd(ctx, s.redisGlobalSetKey(), float64(now), member); err != nil {
		return err
	}

	if err = s.localCache.Set(ctx, s.localKey(userId, modelName), &localSessionKeepValue{AgentId: agentId}, cfg.Ttl); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

func (s *sModelAgentSessionKeep) Refresh(ctx context.Context, userId int, modelName, agentId string) error {
	return s.Set(ctx, userId, modelName, agentId)
}

func (s *sModelAgentSessionKeep) Delete(ctx context.Context, userId int, modelName, agentId string) error {

	if _, err := redis.Del(ctx, s.redisValueKey(userId, modelName), s.redisFailKey(userId, modelName, agentId)); err != nil {
		return err
	}

	if err := s.removeIndex(ctx, userId, modelName, agentId); err != nil {
		return err
	}

	if _, err := s.localCache.Remove(ctx, s.localKey(userId, modelName)); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

func (s *sModelAgentSessionKeep) DeleteByAgent(ctx context.Context, agentId string) (int64, error) {

	members, err := redis.ZRange(ctx, s.redisAgentSetKey(agentId), 0, -1)
	if err != nil {
		return 0, err
	}

	var deleted int64

	for _, member := range members {
		userId, modelName, ok := s.parseMember(member)
		if !ok {
			continue
		}
		if delErr := s.Delete(ctx, userId, modelName, agentId); delErr != nil {
			return deleted, delErr
		}
		deleted++
	}

	if _, err = redis.Del(ctx, s.redisAgentSetKey(agentId)); err != nil {
		return deleted, err
	}

	if deleted > 0 {
		if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			if clearErr := s.localCache.Clear(ctx); clearErr != nil {
				logger.Error(ctx, clearErr)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}

	return deleted, nil
}

func (s *sModelAgentSessionKeep) DeleteAll(ctx context.Context) (int64, error) {

	keys, err := redis.Keys(ctx, "session:agent:*")
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := redis.Del(ctx, keys...)
	if err != nil {
		return 0, err
	}

	if clearErr := s.localCache.Clear(ctx); clearErr != nil {
		logger.Error(ctx, clearErr)
	}

	return deleted, nil
}

func (s *sModelAgentSessionKeep) RecordFail(ctx context.Context, userId int, modelName, agentId string) (int64, error) {

	cfg, err := s.cfgByAgent(ctx, agentId)
	if err != nil {
		return 0, err
	}

	count, err := redis.Incr(ctx, s.redisFailKey(userId, modelName, agentId))
	if err != nil {
		return 0, err
	}

	if _, err = redis.Expire(ctx, s.redisFailKey(userId, modelName, agentId), int64(cfg.FailTtl)); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *sModelAgentSessionKeep) ClearFail(ctx context.Context, userId int, modelName, agentId string) error {
	_, err := redis.Del(ctx, s.redisFailKey(userId, modelName, agentId))
	return err
}

func (s *sModelAgentSessionKeep) ShouldSwitch(ctx context.Context, failCount int64) bool {
	return failCount >= s.cfg().FailSwitchThreshold
}

func (s *sModelAgentSessionKeep) GetAgentCount(ctx context.Context, agentId string) (int64, error) {
	return redis.ZCard(ctx, s.redisAgentSetKey(agentId))
}

func (s *sModelAgentSessionKeep) GetGlobalCount(ctx context.Context) (int64, error) {
	return redis.ZCard(ctx, s.redisGlobalSetKey())
}

func (s *sModelAgentSessionKeep) redisValueKey(userId int, modelName string) string {
	return fmt.Sprintf(sessionKeepValuePrefix, userId, modelName)
}

func (s *sModelAgentSessionKeep) redisFailKey(userId int, modelName, agentId string) string {
	return fmt.Sprintf(sessionKeepFailPrefix, userId, modelName, agentId)
}

func (s *sModelAgentSessionKeep) localKey(userId int, modelName string) string {
	return fmt.Sprintf(sessionKeepLocalKeyPrefix, userId, modelName)
}

func (s *sModelAgentSessionKeep) redisAgentSetKey(agentId string) string {
	return fmt.Sprintf(sessionKeepAgentSetPrefix, agentId)
}

func (s *sModelAgentSessionKeep) redisUserAgentSetKey(userId int, agentId string) string {
	return fmt.Sprintf(sessionKeepUserAgentSet, userId, agentId)
}

func (s *sModelAgentSessionKeep) redisGlobalSetKey() string {
	return sessionKeepGlobalSet
}

func (s *sModelAgentSessionKeep) member(userId int, modelName string) string {
	return fmt.Sprintf("%d:%s", userId, modelName)
}

func (s *sModelAgentSessionKeep) parseMember(member string) (int, string, bool) {

	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 {
		return 0, "", false
	}

	var userId int
	if _, err := fmt.Sscanf(parts[0], "%d", &userId); err != nil || userId <= 0 {
		return 0, "", false
	}

	return userId, parts[1], true
}

func (s *sModelAgentSessionKeep) cfg() *common.ModelAgentSessionKeep {

	if config.Cfg != nil && config.Cfg.SysConfig != nil && config.Cfg.SysConfig.ModelAgentSessionKeep != nil {

		cfg := *config.Cfg.SysConfig.ModelAgentSessionKeep

		if cfg.Ttl <= 0 {
			cfg.Ttl = defaultSessionKeepTTL
		}

		if cfg.FailTtl <= 0 {
			cfg.FailTtl = defaultSessionKeepFailureTTL
		}

		if cfg.FailSwitchThreshold <= 0 {
			cfg.FailSwitchThreshold = defaultSessionKeepSwitchLimit
		}

		if cfg.UserLimit <= 0 {
			cfg.UserLimit = defaultSessionKeepUserLimit
		}

		if cfg.AgentLimit <= 0 {
			cfg.AgentLimit = defaultSessionKeepAgentLimit
		}

		if cfg.GlobalLimit <= 0 {
			cfg.GlobalLimit = defaultSessionKeepGlobalLimit
		}

		return &cfg
	}

	return &common.ModelAgentSessionKeep{
		Open:                false,
		Ttl:                 defaultSessionKeepTTL,
		FailTtl:             defaultSessionKeepFailureTTL,
		FailSwitchThreshold: defaultSessionKeepSwitchLimit,
		UserLimit:           defaultSessionKeepUserLimit,
		AgentLimit:          defaultSessionKeepAgentLimit,
		GlobalLimit:         defaultSessionKeepGlobalLimit,
	}
}

func (s *sModelAgentSessionKeep) cfgByAgent(ctx context.Context, agentId string) (*common.ModelAgentSessionKeep, error) {

	cfg := *s.cfg()
	if agentId == "" {
		return &cfg, nil
	}

	var (
		modelAgent *model.ModelAgent
		err        error
	)

	if modelAgent, err = service.ModelAgent().GetCache(ctx, agentId); err != nil || modelAgent == nil {
		modelAgent, err = service.ModelAgent().GetAndSaveCache(ctx, agentId)
		if err != nil {
			return nil, err
		}
	}

	if modelAgent != nil && modelAgent.IsEnableSessionKeep && modelAgent.SessionKeepConfig != nil {

		agentCfg := modelAgent.SessionKeepConfig

		cfg.Open = true

		if agentCfg.Ttl > 0 {
			cfg.Ttl = agentCfg.Ttl
		}

		if agentCfg.FailTtl > 0 {
			cfg.FailTtl = agentCfg.FailTtl
		}

		if agentCfg.FailSwitchThreshold > 0 {
			cfg.FailSwitchThreshold = agentCfg.FailSwitchThreshold
		}

		if agentCfg.UserLimit > 0 {
			cfg.UserLimit = agentCfg.UserLimit
		}

		if agentCfg.AgentLimit > 0 {
			cfg.AgentLimit = agentCfg.AgentLimit
		}

		if agentCfg.GlobalLimit > 0 {
			cfg.GlobalLimit = agentCfg.GlobalLimit
		}
	}

	return &cfg, nil
}

func (s *sModelAgentSessionKeep) ensureLimit(ctx context.Context, userId int, modelName, agentId string, cfg *common.ModelAgentSessionKeep) error {

	memberLimit := cfg.UserLimit
	if memberLimit > 0 {

		userCount, err := s.compactUserAgentSet(ctx, userId, agentId)
		if err != nil {
			return err
		}

		if userCount >= memberLimit {

			oldModels, err := redis.ZRange(ctx, s.redisUserAgentSetKey(userId, agentId), 0, 0)
			if err != nil {
				return err
			}

			if len(oldModels) > 0 {
				if err = s.Delete(ctx, userId, oldModels[0], agentId); err != nil {
					return err
				}
			}
		}
	}

	if cfg.AgentLimit > 0 {

		agentCount, err := s.compactAgentSet(ctx, agentId)
		if err != nil {
			return err
		}

		if agentCount >= cfg.AgentLimit {

			oldMembers, err := redis.ZRange(ctx, s.redisAgentSetKey(agentId), 0, 0)
			if err != nil {
				return err
			}

			if len(oldMembers) > 0 {

				oldUserId, oldModelName, ok := s.parseMember(oldMembers[0])
				if ok {
					if err = s.Delete(ctx, oldUserId, oldModelName, agentId); err != nil {
						return err
					}
				}
			}
		}
	}

	if cfg.GlobalLimit > 0 {

		globalCount, err := s.compactGlobalSet(ctx)
		if err != nil {
			return err
		}

		if globalCount >= cfg.GlobalLimit {

			oldMembers, err := redis.ZRange(ctx, s.redisGlobalSetKey(), 0, 0)
			if err != nil {
				return err
			}

			if len(oldMembers) > 0 {

				oldUserId, oldModelName, ok := s.parseMember(oldMembers[0])
				if ok {

					oldAgentId, exists, getErr := s.Get(ctx, oldUserId, oldModelName)
					if getErr != nil {
						return getErr
					}

					if exists {
						if err = s.Delete(ctx, oldUserId, oldModelName, oldAgentId); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *sModelAgentSessionKeep) getStoredAgentId(ctx context.Context, userId int, modelName string) (string, bool, error) {

	agentId, err := redis.GetStr(ctx, s.redisValueKey(userId, modelName))
	if err != nil {
		return "", false, err
	}

	if agentId == "" {
		return "", false, nil
	}

	return agentId, true, nil
}

func (s *sModelAgentSessionKeep) removeIndex(ctx context.Context, userId int, modelName, agentId string) error {

	member := s.member(userId, modelName)
	if agentId != "" {

		if _, err := redis.ZRem(ctx, s.redisAgentSetKey(agentId), member); err != nil {
			return err
		}

		if _, err := redis.ZRem(ctx, s.redisUserAgentSetKey(userId, agentId), modelName); err != nil {
			return err
		}
	}

	if _, err := redis.ZRem(ctx, s.redisGlobalSetKey(), member); err != nil {
		return err
	}

	return nil
}

func (s *sModelAgentSessionKeep) cleanupExpired(ctx context.Context, userId int, modelName string) error {

	_, exists, err := s.getStoredAgentId(ctx, userId, modelName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if err = s.removeIndex(ctx, userId, modelName, ""); err != nil {
		return err
	}

	if _, err = s.localCache.Remove(ctx, s.localKey(userId, modelName)); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

func (s *sModelAgentSessionKeep) compactUserAgentSet(ctx context.Context, userId int, agentId string) (int64, error) {

	modelNames, err := redis.ZRange(ctx, s.redisUserAgentSetKey(userId, agentId), 0, -1)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, modelName := range modelNames {

		storedAgentId, exists, getErr := s.getStoredAgentId(ctx, userId, modelName)
		if getErr != nil {
			return 0, getErr
		}

		if !exists || storedAgentId != agentId {
			if remErr := s.removeIndex(ctx, userId, modelName, agentId); remErr != nil {
				return 0, remErr
			}
			continue
		}
		count++
	}

	return count, nil
}

func (s *sModelAgentSessionKeep) compactAgentSet(ctx context.Context, agentId string) (int64, error) {

	members, err := redis.ZRange(ctx, s.redisAgentSetKey(agentId), 0, -1)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, member := range members {

		userId, modelName, ok := s.parseMember(member)
		if !ok {
			continue
		}

		storedAgentId, exists, getErr := s.getStoredAgentId(ctx, userId, modelName)
		if getErr != nil {
			return 0, getErr
		}

		if !exists || storedAgentId != agentId {
			if remErr := s.removeIndex(ctx, userId, modelName, agentId); remErr != nil {
				return 0, remErr
			}
			continue
		}
		count++
	}

	return count, nil
}

func (s *sModelAgentSessionKeep) compactGlobalSet(ctx context.Context) (int64, error) {

	members, err := redis.ZRange(ctx, s.redisGlobalSetKey(), 0, -1)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, member := range members {

		userId, modelName, ok := s.parseMember(member)
		if !ok {
			continue
		}

		_, exists, getErr := s.getStoredAgentId(ctx, userId, modelName)
		if getErr != nil {
			return 0, getErr
		}

		if !exists {
			if remErr := s.removeIndex(ctx, userId, modelName, ""); remErr != nil {
				return 0, remErr
			}
			continue
		}
		count++
	}

	return count, nil
}
