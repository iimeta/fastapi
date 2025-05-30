package key

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/lb"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"slices"
)

type sKey struct {
	modelKeysCache           *cache.Cache // [模型ID][]密钥列表
	modelKeysRoundRobinCache *cache.Cache // [模型ID]密钥下标索引
}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{
		modelKeysRoundRobinCache: cache.New(),
		modelKeysCache:           cache.New(),
	}
}

// 根据secretKey获取密钥信息
func (s *sKey) GetKey(ctx context.Context, secretKey string) (*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := dao.Key.FindOne(ctx, bson.M{"key": secretKey, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Key{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		Corp:                key.Corp,
		Key:                 key.Key,
		Type:                key.Type,
		Weight:              key.Weight,
		Models:              key.Models,
		ModelAgents:         key.ModelAgents,
		IsNeverDisable:      key.IsNeverDisable,
		IsLimitQuota:        key.IsLimitQuota,
		Quota:               key.Quota,
		UsedQuota:           key.UsedQuota,
		QuotaExpiresRule:    key.QuotaExpiresRule,
		QuotaExpiresAt:      key.QuotaExpiresAt,
		QuotaExpiresMinutes: key.QuotaExpiresMinutes,
		IsBindGroup:         key.IsBindGroup,
		Group:               key.Group,
		IpWhitelist:         key.IpWhitelist,
		IpBlacklist:         key.IpBlacklist,
		Status:              key.Status,
	}, nil
}

// 根据模型ID获取密钥列表
func (s *sKey) GetModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "is_agents_only": false, "models": bson.M{"$in": []string{id}}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:                  result.Id,
			UserId:              result.UserId,
			AppId:               result.AppId,
			Corp:                result.Corp,
			Key:                 result.Key,
			Type:                result.Type,
			Weight:              result.Weight,
			Models:              result.Models,
			ModelAgents:         result.ModelAgents,
			IsNeverDisable:      result.IsNeverDisable,
			IsLimitQuota:        result.IsLimitQuota,
			Quota:               result.Quota,
			UsedQuota:           result.UsedQuota,
			QuotaExpiresRule:    result.QuotaExpiresRule,
			QuotaExpiresAt:      result.QuotaExpiresAt,
			QuotaExpiresMinutes: result.QuotaExpiresMinutes,
			IsBindGroup:         result.IsBindGroup,
			Group:               result.Group,
			IpWhitelist:         result.IpWhitelist,
			IpBlacklist:         result.IpBlacklist,
			Status:              result.Status,
		})
	}

	return items, nil
}

// 密钥列表
func (s *sKey) List(ctx context.Context, typ int) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{
		"type": typ,
	}

	results, err := dao.Key.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:                  result.Id,
			UserId:              result.UserId,
			AppId:               result.AppId,
			Corp:                result.Corp,
			Key:                 result.Key,
			Type:                result.Type,
			Weight:              result.Weight,
			Models:              result.Models,
			ModelAgents:         result.ModelAgents,
			IsNeverDisable:      result.IsNeverDisable,
			IsLimitQuota:        result.IsLimitQuota,
			Quota:               result.Quota,
			UsedQuota:           result.UsedQuota,
			QuotaExpiresRule:    result.QuotaExpiresRule,
			QuotaExpiresAt:      result.QuotaExpiresAt,
			QuotaExpiresMinutes: result.QuotaExpiresMinutes,
			IsBindGroup:         result.IsBindGroup,
			Group:               result.Group,
			IpWhitelist:         result.IpWhitelist,
			IpBlacklist:         result.IpBlacklist,
			Status:              result.Status,
			Rid:                 result.Rid,
		})
	}

	return items, nil
}

// 挑选模型密钥
func (s *sKey) PickModelKey(ctx context.Context, m *model.Model) (int, *model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey PickModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		modelKeys  []*model.Key
		roundRobin *lb.RoundRobin
		err        error
	)

	if modelKeysValue := s.modelKeysCache.GetVal(ctx, m.Id); modelKeysValue != nil {
		modelKeys = modelKeysValue.([]*model.Key)
	}

	if len(modelKeys) == 0 {

		if modelKeys, err = s.GetCacheModelKeys(ctx, m.Id); err != nil {

			if modelKeys, err = s.GetModelKeys(ctx, m.Id); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}

			if err = s.SaveCacheModelKeys(ctx, m.Id, modelKeys); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}
		}

		if len(modelKeys) == 0 {
			return 0, nil, errors.ERR_NO_AVAILABLE_KEY
		}

		if err = s.modelKeysCache.Set(ctx, m.Id, modelKeys, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	keyList := make([]*model.Key, 0)
	for _, key := range modelKeys {
		// 过滤被禁用的模型密钥
		if key.Status == 1 {
			keyList = append(keyList, key)
		}
	}

	if len(keyList) == 0 {
		return 0, nil, errors.ERR_NO_AVAILABLE_KEY
	}

	filterKeyList := make([]*model.Key, 0)
	if len(keyList) > 1 {
		errorKeys := service.Session().GetErrorKeys(ctx)
		if len(errorKeys) > 0 {
			for _, key := range keyList {
				// 过滤错误的模型代理密钥
				if !slices.Contains(errorKeys, key.Id) {
					filterKeyList = append(filterKeyList, key)
				}
			}
		} else {
			filterKeyList = keyList
		}
	} else {
		filterKeyList = keyList
	}

	if len(filterKeyList) == 0 {
		return 0, nil, errors.ERR_ALL_KEY
	}

	// 负载策略-权重
	if m.LbStrategy == 2 {
		return len(filterKeyList), lb.NewKeyWeight(filterKeyList).PickKey(), nil
	}

	if roundRobinValue := s.modelKeysRoundRobinCache.GetVal(ctx, m.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*lb.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = lb.NewRoundRobin()
		if err = s.modelKeysRoundRobinCache.Set(ctx, m.Id, roundRobin, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	return len(keyList), keyList[roundRobin.Index(len(keyList))], nil
}

// 移除模型密钥
func (s *sKey) RemoveModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey RemoveModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	keysValue := s.modelKeysCache.GetVal(ctx, m.Id)
	if keysValue != nil {

		if keys := keysValue.([]*model.Key); len(keys) > 0 {

			newKeys := make([]*model.Key, 0)
			for _, k := range keys {
				if k.Id != key.Id {
					newKeys = append(newKeys, k)
				}
			}

			if err := s.modelKeysCache.Set(ctx, m.Id, newKeys, 0); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := dao.Key.UpdateById(ctx, key.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型密钥
func (s *sKey) RecordErrorModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey RecordErrorModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= config.Cfg.Base.ModelKeyErrDisable {
		s.DisabledModelKey(ctx, key, "Reached the maximum number of errors")
	}
}

// 禁用模型密钥
func (s *sKey) DisabledModelKey(ctx context.Context, key *model.Key, disabledReason string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey DisabledModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	// 永不禁用
	if key.IsNeverDisable {
		return
	}

	s.UpdateCacheModelKey(ctx, nil, &entity.Key{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		Corp:                key.Corp,
		Key:                 key.Key,
		Type:                key.Type,
		Weight:              key.Weight,
		Models:              key.Models,
		ModelAgents:         key.ModelAgents,
		IsNeverDisable:      key.IsNeverDisable,
		IsLimitQuota:        key.IsLimitQuota,
		Quota:               key.Quota,
		UsedQuota:           key.UsedQuota,
		QuotaExpiresRule:    key.QuotaExpiresRule,
		QuotaExpiresAt:      key.QuotaExpiresAt,
		QuotaExpiresMinutes: key.QuotaExpiresMinutes,
		IsBindGroup:         key.IsBindGroup,
		Group:               key.Group,
		IpWhitelist:         key.IpWhitelist,
		IpBlacklist:         key.IpBlacklist,
		Status:              2,
		IsAutoDisabled:      true,
		AutoDisabledReason:  disabledReason,
	})

	if err := dao.Key.UpdateById(ctx, key.Id, bson.M{
		"status":               2,
		"is_auto_disabled":     true,
		"auto_disabled_reason": disabledReason,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存模型密钥列表到缓存
func (s *sKey) SaveCacheModelKeys(ctx context.Context, id string, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey SaveCacheModelKeys keys: %d, time: %d", len(keys), gtime.TimestampMilli()-now)
	}()

	if err := s.modelKeysCache.Set(ctx, id, keys, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的模型密钥列表
func (s *sKey) GetCacheModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetCacheModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	if modelKeysCacheValue := s.modelKeysCache.GetVal(ctx, id); modelKeysCacheValue != nil {
		return modelKeysCacheValue.([]*model.Key), nil
	}

	return nil, errors.New("modelKeys is nil")
}

// 新增模型密钥到缓存列表中
func (s *sKey) CreateCacheModelKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey CreateCacheModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	k := &model.Key{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		Corp:                key.Corp,
		Key:                 key.Key,
		Type:                key.Type,
		Weight:              key.Weight,
		Models:              key.Models,
		ModelAgents:         key.ModelAgents,
		IsNeverDisable:      key.IsNeverDisable,
		IsLimitQuota:        key.IsLimitQuota,
		Quota:               key.Quota,
		UsedQuota:           key.UsedQuota,
		QuotaExpiresRule:    key.QuotaExpiresRule,
		QuotaExpiresAt:      key.QuotaExpiresAt,
		QuotaExpiresMinutes: key.QuotaExpiresMinutes,
		IsBindGroup:         key.IsBindGroup,
		Group:               key.Group,
		IpWhitelist:         key.IpWhitelist,
		IpBlacklist:         key.IpBlacklist,
		Status:              key.Status,
	}

	for _, id := range key.Models {

		if keysValue := s.modelKeysCache.GetVal(ctx, id); keysValue != nil {
			if err := s.SaveCacheModelKeys(ctx, id, append(keysValue.([]*model.Key), k)); err != nil {
				logger.Error(ctx, err)
			}
		} else {
			if err := s.SaveCacheModelKeys(ctx, id, []*model.Key{k}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}
}

// 更新缓存中的模型密钥
func (s *sKey) UpdateCacheModelKey(ctx context.Context, oldData *entity.Key, newData *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey UpdateCacheModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	key := &model.Key{
		Id:                  newData.Id,
		UserId:              newData.UserId,
		AppId:               newData.AppId,
		Corp:                newData.Corp,
		Key:                 newData.Key,
		Type:                newData.Type,
		Weight:              newData.Weight,
		Models:              newData.Models,
		ModelAgents:         newData.ModelAgents,
		IsNeverDisable:      newData.IsNeverDisable,
		IsLimitQuota:        newData.IsLimitQuota,
		Quota:               newData.Quota,
		UsedQuota:           newData.UsedQuota,
		QuotaExpiresRule:    newData.QuotaExpiresRule,
		QuotaExpiresAt:      newData.QuotaExpiresAt,
		QuotaExpiresMinutes: newData.QuotaExpiresMinutes,
		IsBindGroup:         newData.IsBindGroup,
		Group:               newData.Group,
		IpWhitelist:         newData.IpWhitelist,
		IpBlacklist:         newData.IpBlacklist,
		Status:              newData.Status,
		IsAutoDisabled:      newData.IsAutoDisabled,
		AutoDisabledReason:  newData.AutoDisabledReason,
	}

	// 用于处理oldData时判断作用
	newModelMap := make(map[string]string)

	for _, id := range newData.Models {

		newModelMap[id] = id

		modelKeys, err := s.GetCacheModelKeys(ctx, id)
		if err != nil {
			logger.Error(ctx, err)
		}

		if len(modelKeys) == 0 {
			if modelKeys, err = s.GetModelKeys(ctx, id); err != nil {
				logger.Error(ctx, err)
				continue
			}
		}

		newKeys := make([]*model.Key, 0)
		// 用于处理新添加了模型时判断作用
		keyMap := make(map[string]*model.Key)

		for _, k := range modelKeys {

			if k.Id != newData.Id {
				newKeys = append(newKeys, k)
				keyMap[k.Id] = k
			} else {
				newKeys = append(newKeys, key)
				keyMap[newData.Id] = key
			}
		}

		if keyMap[newData.Id] == nil {
			newKeys = append(newKeys, key)
		}

		if s.modelKeysCache.ContainsKey(ctx, id) {
			if err = s.SaveCacheModelKeys(ctx, id, newKeys); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	// 将变更后被移除模型的模型密钥移除
	if oldData != nil {

		for _, id := range oldData.Models {

			if newModelMap[id] == "" {

				modelKeys, err := s.GetCacheModelKeys(ctx, id)
				if err != nil {
					if modelKeys, err = s.GetModelKeys(ctx, id); err != nil {
						logger.Error(ctx, err)
						continue
					}
				} else if len(modelKeys) == 0 {
					if modelKeys, err = s.GetModelKeys(ctx, id); err != nil {
						logger.Error(ctx, err)
						continue
					}
				}

				if len(modelKeys) > 0 && s.modelKeysCache.ContainsKey(ctx, id) {

					newKeys := make([]*model.Key, 0)
					for _, k := range modelKeys {
						if k.Id != oldData.Id {
							newKeys = append(newKeys, k)
						}
					}

					if err = s.modelKeysCache.Set(ctx, id, newKeys, 0); err != nil {
						logger.Error(ctx, err)
					}
				}
			}
		}
	}

	if len(newData.ModelAgents) > 0 || (oldData != nil && len(oldData.ModelAgents) > 0) {
		service.ModelAgent().UpdateCacheModelAgentKey(ctx, oldData, newData)
	}
}

// 移除缓存中的模型密钥
func (s *sKey) RemoveCacheModelKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey RemoveCacheModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	for _, id := range key.Models {

		if keysValue := s.modelKeysCache.GetVal(ctx, id); keysValue != nil {

			if keys := keysValue.([]*model.Key); len(keys) > 0 {

				newKeys := make([]*model.Key, 0)
				for _, k := range keys {
					if k.Id != key.Id {
						newKeys = append(newKeys, k)
					}
				}

				if err := s.modelKeysCache.Set(ctx, id, newKeys, 0); err != nil {
					logger.Error(ctx, err)
				}
			}
		}
	}
}

// 密钥已用额度
func (s *sKey) UsedQuota(ctx context.Context, key string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey UsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Key.UpdateOne(ctx, bson.M{"key": key}, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 变更订阅
func (s *sKey) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sKey Subscribe: %s", gjson.MustEncodeString(message))

	var key *entity.Key
	switch message.Action {
	case consts.ACTION_CREATE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if key.IsAgentsOnly {
			service.ModelAgent().CreateCacheModelAgentKey(ctx, key)
		} else {
			s.CreateCacheModelKey(ctx, key)
			service.ModelAgent().CreateCacheModelAgentKey(ctx, key)
		}

	case consts.ACTION_UPDATE, consts.ACTION_MODELS:

		var oldData *entity.Key
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &oldData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if !oldData.IsAgentsOnly && !key.IsAgentsOnly {
			s.UpdateCacheModelKey(ctx, oldData, key)
		} else if oldData.IsAgentsOnly && key.IsAgentsOnly {
			service.ModelAgent().UpdateCacheModelAgentKey(ctx, oldData, key)
		} else if oldData.IsAgentsOnly && !key.IsAgentsOnly {
			s.CreateCacheModelKey(ctx, key)
			service.ModelAgent().UpdateCacheModelAgentKey(ctx, oldData, key)
		} else if !oldData.IsAgentsOnly && key.IsAgentsOnly {
			s.RemoveCacheModelKey(ctx, oldData)
			service.ModelAgent().UpdateCacheModelAgentKey(ctx, oldData, key)
		} else { // 似乎永远都走不了这个
			s.UpdateCacheModelKey(ctx, oldData, key)
			service.ModelAgent().UpdateCacheModelAgentKey(ctx, oldData, key)
		}

	case consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if key.Type == 1 {
			service.App().UpdateCacheAppKey(ctx, key)
		} else {
			if key.IsAgentsOnly {
				service.ModelAgent().UpdateCacheModelAgentKey(ctx, nil, key)
			} else {
				s.UpdateCacheModelKey(ctx, nil, key)
				service.ModelAgent().UpdateCacheModelAgentKey(ctx, nil, key)
			}
		}

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if key.Type == 1 {
			service.App().RemoveCacheAppKey(ctx, key.Key)
		} else {
			if key.IsAgentsOnly {
				service.ModelAgent().RemoveCacheModelAgentKey(ctx, key)
			} else {
				s.RemoveCacheModelKey(ctx, key)
				service.ModelAgent().RemoveCacheModelAgentKey(ctx, key)
			}
		}
	}

	return nil
}
