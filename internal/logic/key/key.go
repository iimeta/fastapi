package key

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type sKey struct{}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{}
}

// 密钥列表
func (s *sKey) List(ctx context.Context) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey List time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{}, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:             result.Id,
			ProviderId:     result.ProviderId,
			Key:            result.Key,
			Weight:         result.Weight,
			ModelAgents:    result.ModelAgents,
			IsNeverDisable: result.IsNeverDisable,
			UsedQuota:      result.UsedQuota,
			Status:         result.Status,
		})
	}

	return items, nil
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

		service.ModelAgent().CreateCacheKey(ctx, key)

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

		service.ModelAgent().UpdateCacheKey(ctx, oldData, key)

	case consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		service.ModelAgent().UpdateCacheKey(ctx, nil, key)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		service.ModelAgent().RemoveCacheKey(ctx, key)
	}

	return nil
}
