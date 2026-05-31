package session_keep

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
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
	KeyId   string
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

// 获取会话保持绑定的代理和密钥
func (s *sSessionKeepModelAgent) Get(ctx context.Context, sk *common.SessionKey) (string, string, bool, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent Get time: %d", gtime.TimestampMilli()-now)
	}()

	key := s.localKey(sk)
	if value := s.localCache.GetVal(ctx, key); value != nil {
		if localValue, ok := value.(*localSessionKeepValue); ok && localValue.AgentId != "" {
			return localValue.AgentId, localValue.KeyId, true, nil
		}
	}

	agentId, keyId, exists, err := s.getStoredValue(ctx, sk)
	if err != nil {
		return "", "", false, err
	}

	if !exists {
		return "", "", false, nil
	}

	cfg, cfgErr := s.cfgByAgent(ctx, agentId)
	if cfgErr != nil {
		return "", "", false, cfgErr
	}

	localTtl := s.localTTL(cfg.Ttl)
	if err = s.localCache.Set(ctx, key, &localSessionKeepValue{AgentId: agentId, KeyId: keyId}, localTtl); err != nil {
		logger.Error(ctx, err)
	}

	return agentId, keyId, true, nil
}

// 设置会话保持绑定
func (s *sSessionKeepModelAgent) Set(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent Set time: %d", gtime.TimestampMilli()-now)
	}()

	if agentId == "" || sk == nil || sk.UserId <= 0 {
		return nil
	}

	if err := s.cleanupExpired(ctx, sk); err != nil {
		return err
	}

	cfg, err := s.cfgByAgent(ctx, agentId)
	if err != nil {
		return err
	}

	oldAgentId, _, exists, err := s.getStoredValue(ctx, sk)
	if err != nil {
		return err
	}

	if exists && oldAgentId != "" && oldAgentId != agentId {
		if err = s.removeIndex(ctx, sk, oldAgentId); err != nil {
			return err
		}
	}

	if err = s.ensureLimit(ctx, sk, agentId, cfg); err != nil {
		return err
	}

	value := agentId
	if keyId != "" {
		value = agentId + ":" + keyId
	}

	if err = redis.SetEX(ctx, s.redisValueKey(sk), value, s.redisTTLSeconds(cfg.Ttl)); err != nil {
		return err
	}

	ts := time.Now().Unix()
	member := sk.Raw
	if _, err := redis.ZAdd(ctx, s.redisAgentSetKey(agentId), float64(ts), member); err != nil {
		return err
	}

	if _, err := redis.ZAdd(ctx, s.redisUserAgentSetKey(sk.UserId, agentId), float64(ts), member); err != nil {
		return err
	}

	if _, err := redis.ZAdd(ctx, s.redisGlobalSetKey(), float64(ts), member); err != nil {
		return err
	}

	localTtl := s.localTTL(cfg.Ttl)
	if err = s.localCache.Set(ctx, s.localKey(sk), &localSessionKeepValue{AgentId: agentId, KeyId: keyId}, localTtl); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 刷新会话保持绑定
func (s *sSessionKeepModelAgent) Refresh(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent Refresh time: %d", gtime.TimestampMilli()-now)
	}()

	return s.Set(ctx, sk, agentId, keyId)
}

// 删除会话保持绑定
func (s *sSessionKeepModelAgent) Delete(ctx context.Context, sk *common.SessionKey, agentId string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent Delete time: %d", gtime.TimestampMilli()-now)
	}()

	_, keyId, _, _ := s.getStoredValue(ctx, sk)

	keysToDelete := []string{s.redisValueKey(sk), s.redisFailKey(sk, agentId)}
	if keyId != "" {
		keysToDelete = append(keysToDelete, s.redisKeyFailKey(sk, agentId, keyId))
	}

	if _, err := redis.Del(ctx, keysToDelete...); err != nil {
		return err
	}

	if err := s.removeIndex(ctx, sk, agentId); err != nil {
		return err
	}

	if _, err := s.localCache.Remove(ctx, s.localKey(sk)); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 根据代理删除所有会话保持绑定
func (s *sSessionKeepModelAgent) DeleteByAgent(ctx context.Context, agentId string) (int64, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent DeleteByAgent time: %d", gtime.TimestampMilli()-now)
	}()

	members, err := redis.ZRange(ctx, s.redisAgentSetKey(agentId), 0, -1)
	if err != nil {
		return 0, err
	}

	var deleted int64

	for _, member := range members {
		sk := recoverSessionKey(member)
		if sk == nil {
			continue
		}
		if delErr := s.Delete(ctx, sk, agentId); delErr != nil {
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

// 删除所有会话保持绑定
func (s *sSessionKeepModelAgent) DeleteAll(ctx context.Context) (int64, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent DeleteAll time: %d", gtime.TimestampMilli()-now)
	}()

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

// 记录代理失败次数
func (s *sSessionKeepModelAgent) RecordFail(ctx context.Context, sk *common.SessionKey, agentId string) (int64, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent RecordFail time: %d", gtime.TimestampMilli()-now)
	}()

	cfg, err := s.cfgByAgent(ctx, agentId)
	if err != nil {
		return 0, err
	}

	failKey := s.redisFailKey(sk, agentId)

	count, err := redis.Incr(ctx, failKey)
	if err != nil {
		return 0, err
	}

	if _, err = redis.Expire(ctx, failKey, s.redisTTLSeconds(cfg.FailTtl)); err != nil {
		return 0, err
	}

	return count, nil
}

// 清除代理失败计数
func (s *sSessionKeepModelAgent) ClearFail(ctx context.Context, sk *common.SessionKey, agentId string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent ClearFail time: %d", gtime.TimestampMilli()-now)
	}()

	_, err := redis.Del(ctx, s.redisFailKey(sk, agentId))

	return err
}

// 记录密钥失败次数
func (s *sSessionKeepModelAgent) RecordKeyFail(ctx context.Context, sk *common.SessionKey, agentId, keyId string) (int64, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent RecordKeyFail time: %d", gtime.TimestampMilli()-now)
	}()

	cfg, err := s.cfgByAgent(ctx, agentId)
	if err != nil {
		return 0, err
	}

	failKey := s.redisKeyFailKey(sk, agentId, keyId)

	count, err := redis.Incr(ctx, failKey)
	if err != nil {
		return 0, err
	}

	if _, err = redis.Expire(ctx, failKey, s.redisTTLSeconds(cfg.FailTtl)); err != nil {
		return 0, err
	}

	return count, nil
}

// 清除密钥失败计数
func (s *sSessionKeepModelAgent) ClearKeyFail(ctx context.Context, sk *common.SessionKey, agentId, keyId string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sSessionKeepModelAgent ClearKeyFail time: %d", gtime.TimestampMilli()-now)
	}()

	_, err := redis.Del(ctx, s.redisKeyFailKey(sk, agentId, keyId))

	return err
}

func (s *sSessionKeepModelAgent) redisValueKey(sk *common.SessionKey) string {
	return fmt.Sprintf(consts.SESSION_KEEP_VALUE_PREFIX, sk.Raw)
}

func (s *sSessionKeepModelAgent) redisFailKey(sk *common.SessionKey, agentId string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_FAIL_PREFIX, sk.Raw, agentId)
}

func (s *sSessionKeepModelAgent) redisKeyFailKey(sk *common.SessionKey, agentId, keyId string) string {
	return fmt.Sprintf(consts.SESSION_KEEP_KEY_FAIL_PREFIX, sk.Raw, agentId, keyId)
}

func (s *sSessionKeepModelAgent) localKey(sk *common.SessionKey) string {
	return fmt.Sprintf(consts.SESSION_KEEP_LOCAL_KEY_PREFIX, sk.Raw)
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

		if agentCfg.Mode != "" {
			cfg.Mode = agentCfg.Mode
		}

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

		if len(agentCfg.Rules) > 0 {
			cfg.Rules = agentCfg.Rules
		}

		if agentCfg.EnableSystemPromptHash {
			cfg.EnableSystemPromptHash = true
		}
	}

	return &cfg, nil
}

func (s *sSessionKeepModelAgent) ensureLimit(ctx context.Context, sk *common.SessionKey, agentId string, cfg *common.ModelAgentSessionKeep) error {

	memberLimit := cfg.UserLimit
	if memberLimit > 0 {

		userCount, err := s.compactUserAgentSet(ctx, sk.UserId, agentId)
		if err != nil {
			return err
		}

		if userCount >= memberLimit {

			oldMembers, err := redis.ZRange(ctx, s.redisUserAgentSetKey(sk.UserId, agentId), 0, 0)
			if err != nil {
				return err
			}

			if len(oldMembers) > 0 {
				oldSk := recoverSessionKey(oldMembers[0])
				if oldSk != nil {
					if err = s.Delete(ctx, oldSk, agentId); err != nil {
						return err
					}
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
				oldSk := recoverSessionKey(oldMembers[0])
				if oldSk != nil {
					if err = s.Delete(ctx, oldSk, agentId); err != nil {
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
				oldSk := recoverSessionKey(oldMembers[0])
				if oldSk != nil {

					oldAgentId, _, exists, getErr := s.Get(ctx, oldSk)
					if getErr != nil {
						return getErr
					}

					if exists {
						if err = s.Delete(ctx, oldSk, oldAgentId); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *sSessionKeepModelAgent) getStoredValue(ctx context.Context, sk *common.SessionKey) (string, string, bool, error) {

	val, err := redis.GetStr(ctx, s.redisValueKey(sk))
	if err != nil {
		return "", "", false, err
	}

	if val == "" {
		return "", "", false, nil
	}

	parts := strings.SplitN(val, ":", 2)
	agentId := parts[0]
	keyId := ""
	if len(parts) == 2 {
		keyId = parts[1]
	}

	return agentId, keyId, true, nil
}

func (s *sSessionKeepModelAgent) removeIndex(ctx context.Context, sk *common.SessionKey, agentId string) error {

	member := sk.Raw
	if agentId != "" {

		if _, err := redis.ZRem(ctx, s.redisAgentSetKey(agentId), member); err != nil {
			return err
		}

		if _, err := redis.ZRem(ctx, s.redisUserAgentSetKey(sk.UserId, agentId), member); err != nil {
			return err
		}
	}

	if _, err := redis.ZRem(ctx, s.redisGlobalSetKey(), member); err != nil {
		return err
	}

	return nil
}

func (s *sSessionKeepModelAgent) cleanupExpired(ctx context.Context, sk *common.SessionKey) error {

	_, _, exists, err := s.getStoredValue(ctx, sk)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if err = s.removeIndex(ctx, sk, ""); err != nil {
		return err
	}

	if _, err = s.localCache.Remove(ctx, s.localKey(sk)); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

func (s *sSessionKeepModelAgent) compactUserAgentSet(ctx context.Context, userId int, agentId string) (int64, error) {

	members, err := redis.ZRange(ctx, s.redisUserAgentSetKey(userId, agentId), 0, -1)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, member := range members {

		sk := recoverSessionKey(member)
		if sk == nil {
			if _, remErr := redis.ZRem(ctx, s.redisUserAgentSetKey(userId, agentId), member); remErr != nil {
				return 0, remErr
			}
			continue
		}

		storedAgentId, _, exists, getErr := s.getStoredValue(ctx, sk)
		if getErr != nil {
			return 0, getErr
		}

		if !exists || storedAgentId != agentId {
			if remErr := s.removeIndex(ctx, sk, agentId); remErr != nil {
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

		sk := recoverSessionKey(member)
		if sk == nil {
			if _, remErr := redis.ZRem(ctx, s.redisAgentSetKey(agentId), member); remErr != nil {
				return 0, remErr
			}
			continue
		}

		storedAgentId, _, exists, getErr := s.getStoredValue(ctx, sk)
		if getErr != nil {
			return 0, getErr
		}

		if !exists || storedAgentId != agentId {
			if remErr := s.removeIndex(ctx, sk, agentId); remErr != nil {
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

		sk := recoverSessionKey(member)
		if sk == nil {
			if _, remErr := redis.ZRem(ctx, s.redisGlobalSetKey(), member); remErr != nil {
				return 0, remErr
			}
			continue
		}

		_, _, exists, getErr := s.getStoredValue(ctx, sk)
		if getErr != nil {
			return 0, getErr
		}

		if !exists {
			if remErr := s.removeIndex(ctx, sk, ""); remErr != nil {
				return 0, remErr
			}
			continue
		}
		count++
	}

	return count, nil
}

func (s *sSessionKeepModelAgent) localTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	return time.Second * time.Duration(s.redisTTLSeconds(ttl))
}

func (s *sSessionKeepModelAgent) redisTTLSeconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		return int64(ttl)
	}
	return int64(ttl)
}
