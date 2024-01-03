package key

import (
	"context"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sKey struct{}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{}
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

// 根据model获取密钥信息
func (s *sKey) GetModelKey(ctx context.Context, m string) (*model.Key, error) {

	key, err := dao.Key.FindOne(ctx, bson.M{"models": bson.M{"$in": []string{m}}})
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

// 密钥列表
func (s *sKey) List(ctx context.Context) ([]*model.Key, error) {

	filter := bson.M{
		"type": 1,
	}

	results, err := dao.Key.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:     result.Id,
			Corp:   result.Corp,
			Key:    result.Key,
			Type:   result.Type,
			Models: result.Models,
			Remark: result.Remark,
			Status: result.Status,
		})
	}

	return items, nil
}
