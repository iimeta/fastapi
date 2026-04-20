package session_keep

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
)

type localSessionKeepValue struct {
	AgentId string
}

type sSessionKeepModelAgent struct {
	localCache *cache.Cache
}

func init() {
	service.RegisterSessionKeepModelAgent(New())
}

func New() service.ISessionKeepModelAgent {
	return &sSessionKeepModelAgent{
		localCache: cache.New(),
	}
}

func (s *sSessionKeepModelAgent) Get(ctx context.Context, userId int, modelName string) (string, bool, error) {

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

func (s *sSessionKeepModelAgent) Set(ctx context.Context, userId int, modelName, agentId string) error {

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

func (s *sSessionKeepModelAgent) Refresh(ctx context.Context, userId int, modelName, agentId string) error {
	return s.Set(ctx, userId, modelName, agentId)
}

func (s *sSessionKeepModelAgent) Delete(ctx context.Context, userId int, modelName, agentId string) error {

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

func (s *sSessionKeepModelAgent) DeleteByAgent(ctx context.Context, agentId string) (int64, error) {

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

func (s *sSessionKeepModelAgent) DeleteAll(ctx context.Context) (int64, error) {

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

func (s *sSessionKeepModelAgent) RecordFail(ctx context.Context, userId int, modelName, agentId string) (int64, error) {

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

func (s *sSessionKeepModelAgent) ClearFail(ctx context.Context, userId int, modelName, agentId string) error {
	_, err := redis.Del(ctx, s.redisFailKey(userId, modelName, agentId))
	return err
}

func (s *sSessionKeepModelAgent) GetAgentCount(ctx context.Context, agentId string) (int64, error) {
	return redis.ZCard(ctx, s.redisAgentSetKey(agentId))
}

func (s *sSessionKeepModelAgent) GetGlobalCount(ctx context.Context) (int64, error) {
	return redis.ZCard(ctx, s.redisGlobalSetKey())
}

func (s *sSessionKeepModelAgent) redisValueKey(userId int, modelName string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_VALUE_PREFIX, userId, modelName)
}

func (s *sSessionKeepModelAgent) redisFailKey(userId int, modelName, agentId string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_FAIL_PREFIX, userId, modelName, agentId)
}

func (s *sSessionKeepModelAgent) localKey(userId int, modelName string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_LOCAL_KEY_PREFIX, userId, modelName)
}

func (s *sSessionKeepModelAgent) redisAgentSetKey(agentId string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_AGENT_SET_PREFIX, agentId)
}

func (s *sSessionKeepModelAgent) redisUserAgentSetKey(userId int, agentId string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_USER_AGENT_SET, userId, agentId)
}

func (s *sSessionKeepModelAgent) redisGlobalSetKey() string {
	return consts.SESSION_KEEP_GLOBAL_SET
}

func (s *sSessionKeepModelAgent) member(userId int, modelName string) string {
	return fmt.Sprintf("%d:%s", userId, modelName)
}

func (s *sSessionKeepModelAgent) parseMember(member string) (int, string, bool) {

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

func (s *sSessionKeepModelAgent) cfgByAgent(ctx context.Context, agentId string) (*common.ModelAgentSessionKeep, error) {

	cfg := *config.Cfg.SysConfig.ModelAgentSessionKeep
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

func (s *sSessionKeepModelAgent) ensureLimit(ctx context.Context, userId int, modelName, agentId string, cfg *common.ModelAgentSessionKeep) error {

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

func (s *sSessionKeepModelAgent) getStoredAgentId(ctx context.Context, userId int, modelName string) (string, bool, error) {

	agentId, err := redis.GetStr(ctx, s.redisValueKey(userId, modelName))
	if err != nil {
		return "", false, err
	}

	if agentId == "" {
		return "", false, nil
	}

	return agentId, true, nil
}

func (s *sSessionKeepModelAgent) removeIndex(ctx context.Context, userId int, modelName, agentId string) error {

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

func (s *sSessionKeepModelAgent) cleanupExpired(ctx context.Context, userId int, modelName string) error {

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

func (s *sSessionKeepModelAgent) compactUserAgentSet(ctx context.Context, userId int, agentId string) (int64, error) {

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

func (s *sSessionKeepModelAgent) compactAgentSet(ctx context.Context, agentId string) (int64, error) {

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

func (s *sSessionKeepModelAgent) compactGlobalSet(ctx context.Context) (int64, error) {

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
