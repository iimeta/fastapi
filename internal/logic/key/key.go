package key

import (
	"context"
	"errors"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
	"go.mongodb.org/mongo-driver/bson"
)

type sKey struct {
	keysMap       *gmap.StrAnyMap
	roundRobinMap *gmap.StrAnyMap
}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{
		keysMap:       gmap.NewStrAnyMap(true),
		roundRobinMap: gmap.NewStrAnyMap(true),
	}
}

// 根据secretKey获取密钥信息
func (s *sKey) GetKey(ctx context.Context, secretKey string) (*model.Key, error) {

	key, err := dao.Key.FindOne(ctx, bson.M{"key": secretKey})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Key{
		Id:          key.Id,
		AppId:       key.AppId,
		Corp:        key.Corp,
		Key:         key.Key,
		Type:        key.Type,
		Models:      key.Models,
		Quota:       key.Quota,
		IpWhitelist: key.IpWhitelist,
		IpBlacklist: key.IpBlacklist,
		Remark:      key.Remark,
		Status:      key.Status,
	}, nil
}

// 根据模型ID获取密钥列表
func (s *sKey) GetModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "models": bson.M{"$in": []string{id}}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:          result.Id,
			AppId:       result.AppId,
			Corp:        result.Corp,
			Key:         result.Key,
			Type:        result.Type,
			Models:      result.Models,
			Quota:       result.Quota,
			IpWhitelist: result.IpWhitelist,
			IpBlacklist: result.IpBlacklist,
			Remark:      result.Remark,
			Status:      result.Status,
		})
	}

	return items, nil
}

// 密钥列表
func (s *sKey) List(ctx context.Context, Type int) ([]*model.Key, error) {

	filter := bson.M{
		"type": Type,
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

// 根据模型ID挑选密钥
func (s *sKey) PickModelKey(ctx context.Context, id string) (key *model.Key, err error) {

	var keys []*model.Key
	var roundRobin *util.RoundRobin

	keysValue := s.keysMap.Get(id)
	roundRobinValue := s.roundRobinMap.Get(id)

	if keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		keys, err = s.GetModelKeys(ctx, id)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if len(keys) == 0 {
			return nil, errors.New("当前模型暂无可用密钥")
		}

		s.keysMap.Set(id, keys)
	}

	if roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.roundRobinMap.Set(id, roundRobin)
	}

	return keys[roundRobin.Index(len(keys))], nil
}
