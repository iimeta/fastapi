package key

import (
	"cmp"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"go.mongodb.org/mongo-driver/bson"
	"slices"
)

type sKey struct {
	modelKeysMap           *gmap.StrAnyMap // map[模型ID][]密钥列表
	modelKeysRoundRobinMap *gmap.StrAnyMap // 模型ID->密钥下标索引
	modelKeysCacheMap      *gmap.StrAnyMap // map[模型ID][]密钥列表
}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{
		modelKeysMap:           gmap.NewStrAnyMap(true),
		modelKeysRoundRobinMap: gmap.NewStrAnyMap(true),
		modelKeysCacheMap:      gmap.NewStrAnyMap(true),
	}
}

// 根据secretKey获取密钥信息
func (s *sKey) GetKey(ctx context.Context, secretKey string) (*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := dao.Key.FindOne(ctx, bson.M{"key": secretKey, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Key{
		Id:           key.Id,
		AppId:        key.AppId,
		Corp:         key.Corp,
		Key:          key.Key,
		Type:         key.Type,
		Models:       key.Models,
		IsLimitQuota: key.IsLimitQuota,
		Quota:        key.Quota,
		IpWhitelist:  key.IpWhitelist,
		IpBlacklist:  key.IpBlacklist,
		Remark:       key.Remark,
		Status:       key.Status,
	}, nil
}

// 根据模型ID获取密钥列表
func (s *sKey) GetModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "is_agents_only": false, "status": 1, "models": bson.M{"$in": []string{id}}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:           result.Id,
			AppId:        result.AppId,
			Corp:         result.Corp,
			Key:          result.Key,
			Type:         result.Type,
			Models:       result.Models,
			IsLimitQuota: result.IsLimitQuota,
			Quota:        result.Quota,
			IpWhitelist:  result.IpWhitelist,
			IpBlacklist:  result.IpBlacklist,
			Remark:       result.Remark,
			Status:       result.Status,
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
		"type":   typ,
		"status": 1,
	}

	results, err := dao.Key.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:           result.Id,
			AppId:        result.AppId,
			Corp:         result.Corp,
			Key:          result.Key,
			Type:         result.Type,
			Models:       result.Models,
			IsLimitQuota: result.IsLimitQuota,
			Quota:        result.Quota,
			IpWhitelist:  result.IpWhitelist,
			IpBlacklist:  result.IpBlacklist,
			Remark:       result.Remark,
			Status:       result.Status,
		})
	}

	return items, nil
}

// 挑选模型密钥
func (s *sKey) PickModelKey(ctx context.Context, m *model.Model) (key *model.Key, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "PickModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	var modelKeys []*model.Key
	var roundRobin *util.RoundRobin

	if modelKeysValue := s.modelKeysMap.Get(m.Id); modelKeysValue != nil {
		modelKeys = modelKeysValue.([]*model.Key)
	}

	if len(modelKeys) == 0 {

		if modelKeys, err = s.GetCacheModelKeys(ctx, m.Id); err != nil {

			if modelKeys, err = s.GetModelKeys(ctx, m.Id); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			if err = s.SaveCacheModelKeys(ctx, m.Id, modelKeys); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		if len(modelKeys) == 0 {
			return nil, errors.ERR_NO_AVAILABLE_KEY
		}

		s.modelKeysMap.Set(m.Id, modelKeys)
	}

	keyList := make([]*model.Key, 0)
	for _, key := range modelKeys {
		// 过滤被禁用的模型密钥
		if key.Status == 1 {
			keyList = append(keyList, key)
		}
	}

	if roundRobinValue := s.modelKeysRoundRobinMap.Get(m.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.modelKeysRoundRobinMap.Set(m.Id, roundRobin)
	}

	return keyList[roundRobin.Index(len(keyList))], nil
}

// 移除模型密钥
func (s *sKey) RemoveModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	keysValue := s.modelKeysMap.Get(m.Id)
	if keysValue != nil {

		if keys := keysValue.([]*model.Key); len(keys) > 0 {

			newKeys := make([]*model.Key, 0)
			for _, k := range keys {
				if k.Id != key.Id {
					newKeys = append(newKeys, k)
				}
			}

			s.modelKeysMap.Set(m.Id, newKeys)
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
		logger.Debugf(ctx, "RecordErrorModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {
		s.RemoveModelKey(ctx, m, key)
	}
}

// 更改密钥额度
func (s *sKey) ChangeQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey ChangeQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存模型密钥列表到缓存
func (s *sKey) SaveCacheModelKeys(ctx context.Context, id string, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey SaveCacheModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, key := range keys {
		fields[key.Id] = key
	}

	if len(fields) > 0 {

		if _, err := redis.HSet(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id), fields); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.modelKeysCacheMap.Set(id, keys)
	}

	return nil
}

// 获取缓存中的模型密钥列表
func (s *sKey) GetCacheModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetCacheModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	if modelKeysCacheValue := s.modelKeysCacheMap.Get(id); modelKeysCacheValue != nil {
		return modelKeysCacheValue.([]*model.Key), nil
	}

	reply, err := redis.HVals(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		return nil, errors.New("modelKeys is nil")
	}

	items := make([]*model.Key, 0)
	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.Key)
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if result.Status == 1 {
			items = append(items, result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("modelKeys is nil")
	}

	slices.SortFunc(items, func(k1, k2 *model.Key) int {
		return cmp.Compare(k1.Id, k2.Id)
	})

	s.modelKeysCacheMap.Set(id, items)

	return items, nil
}

// 新增模型密钥到缓存列表中
func (s *sKey) CreateCacheModelKey(ctx context.Context, key *entity.Key) {

	k := &model.Key{
		Id:           key.Id,
		UserId:       key.UserId,
		AppId:        key.AppId,
		Corp:         key.Corp,
		Key:          key.Key,
		Type:         key.Type,
		Models:       key.Models,
		ModelAgents:  key.ModelAgents,
		IsLimitQuota: key.IsLimitQuota,
		Quota:        key.Quota,
		RPM:          key.RPM,
		RPD:          key.RPD,
		IpWhitelist:  key.IpWhitelist,
		IpBlacklist:  key.IpBlacklist,
		Status:       key.Status,
	}

	for _, id := range key.Models {

		if err := s.SaveCacheModelKeys(ctx, id, []*model.Key{k}); err != nil {
			logger.Error(ctx, err)
		}

		if keysValue := s.modelKeysMap.Get(id); keysValue != nil {
			s.modelKeysMap.Set(id, append(keysValue.([]*model.Key), k))
		}
	}
}

// 更新缓存中的模型密钥
func (s *sKey) UpdateCacheModelKey(ctx context.Context, oldData *entity.Key, newData *entity.Key) {

	key := &model.Key{
		Id:           newData.Id,
		UserId:       newData.UserId,
		AppId:        newData.AppId,
		Corp:         newData.Corp,
		Key:          newData.Key,
		Type:         newData.Type,
		Models:       newData.Models,
		ModelAgents:  newData.ModelAgents,
		IsLimitQuota: newData.IsLimitQuota,
		Quota:        newData.Quota,
		RPM:          newData.RPM,
		RPD:          newData.RPD,
		IpWhitelist:  newData.IpWhitelist,
		IpBlacklist:  newData.IpBlacklist,
		Status:       newData.Status,
	}

	// 用于处理oldData时判断作用
	newModelMap := make(map[string]string)

	for _, id := range newData.Models {

		newModelMap[id] = id

		if keysValue := s.modelKeysMap.Get(id); keysValue != nil {

			keys := keysValue.([]*model.Key)
			newKeys := make([]*model.Key, 0)
			// 用于处理新添加了模型时判断作用
			keyMap := make(map[string]*model.Key)

			for _, k := range keys {

				if k.Id != newData.Id {
					newKeys = append(newKeys, k)
					keyMap[key.Id] = k
				} else {
					newKeys = append(newKeys, key)
					keyMap[newData.Id] = key
				}
			}

			if keyMap[newData.Id] == nil {
				newKeys = append(newKeys, key)
			}

			if err := s.SaveCacheModelKeys(ctx, id, newKeys); err != nil {
				logger.Error(ctx, err)
			}

			s.modelKeysMap.Set(id, newKeys)
		}
	}

	// 将变更后被移除模型的模型密钥移除
	if oldData != nil {
		for _, id := range oldData.Models {

			if newModelMap[id] == "" {

				if keysValue := s.modelKeysMap.Get(id); keysValue != nil {

					if keys := keysValue.([]*model.Key); len(keys) > 0 {

						newKeys := make([]*model.Key, 0)
						for _, k := range keys {

							if k.Id != oldData.Id {
								newKeys = append(newKeys, k)
							} else {
								if _, err := redis.HDel(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id), oldData.Id); err != nil {
									logger.Error(ctx, err)
								}
							}
						}

						s.modelKeysMap.Set(id, newKeys)
					}
				}
			}
		}
	}
}

// 移除缓存中的模型密钥
func (s *sKey) RemoveCacheModelKey(ctx context.Context, key *entity.Key) {

	for _, id := range key.Models {

		if keysValue := s.modelKeysMap.Get(id); keysValue != nil {

			if keys := keysValue.([]*model.Key); len(keys) > 0 {

				newKeys := make([]*model.Key, 0)
				for _, k := range keys {

					if k.Id != key.Id {
						newKeys = append(newKeys, k)
					} else {
						if _, err := redis.HDel(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id), key.Id); err != nil {
							logger.Error(ctx, err)
						}
					}
				}

				s.modelKeysMap.Set(id, newKeys)
			}
		}
	}
}

// 变更订阅
func (s *sKey) Subscribe(ctx context.Context, msg string) error {

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

	case consts.ACTION_UPDATE:

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
			service.Common().UpdateCacheAppKey(ctx, key)
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
			service.Common().RemoveCacheAppKey(ctx, key.Key)
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
