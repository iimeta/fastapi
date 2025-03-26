package user

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sUser struct {
	userCache      *cache.Cache // [userId]User
	userQuotaCache *cache.Cache // [userId]Quota
}

func init() {
	service.RegisterUser(New())
}

func New() service.IUser {
	return &sUser{
		userCache:      cache.New(),
		userQuotaCache: cache.New(),
	}
}

// 根据用户ID获取用户信息
func (s *sUser) GetUser(ctx context.Context, userId int) (*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser GetUser time: %d", gtime.TimestampMilli()-now)
	}()

	user, err := dao.User.FindUserByUserId(ctx, userId)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.User{
		Id:             user.Id,
		UserId:         user.UserId,
		Name:           user.Name,
		Avatar:         user.Avatar,
		Email:          user.Email,
		Phone:          user.Phone,
		Quota:          user.Quota,
		UsedQuota:      user.UsedQuota,
		QuotaExpiresAt: user.QuotaExpiresAt,
		Models:         user.Models,
		Status:         user.Status,
	}, nil
}

// 用户列表
func (s *sUser) List(ctx context.Context) ([]*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.User.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.User, 0)
	for _, result := range results {
		items = append(items, &model.User{
			Id:             result.Id,
			UserId:         result.UserId,
			Name:           result.Name,
			Avatar:         result.Avatar,
			Email:          result.Email,
			Phone:          result.Phone,
			Quota:          result.Quota,
			UsedQuota:      result.UsedQuota,
			QuotaExpiresAt: result.QuotaExpiresAt,
			Models:         result.Models,
			Status:         result.Status,
		})
	}

	return items, nil
}

// 用户花费额度
func (s *sUser) SpendQuota(ctx context.Context, userId, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.User.UpdateOne(ctx, bson.M{"user_id": userId}, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存用户信息到缓存
func (s *sUser) SaveCacheUser(ctx context.Context, user *model.User) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser SaveCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if user == nil {
		return errors.New("user is nil")
	}

	service.Session().SaveUser(ctx, user)

	if err := s.userCache.Set(ctx, user.UserId, user, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.userQuotaCache.Set(ctx, user.UserId, user.Quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的用户信息
func (s *sUser) GetCacheUser(ctx context.Context, userId int) (*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser GetCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if user := service.Session().GetUser(ctx); user != nil {
		return user, nil
	}

	if userCacheValue := s.userCache.GetVal(ctx, userId); userCacheValue != nil {
		user := userCacheValue.(*model.User)
		service.Session().SaveUser(ctx, user)
		return user, nil
	}

	return nil, errors.New("user is nil")
}

// 更新缓存中的用户信息
func (s *sUser) UpdateCacheUser(ctx context.Context, user *entity.User) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser UpdateCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheUser(ctx, &model.User{
		Id:             user.Id,
		UserId:         user.UserId,
		Name:           user.Name,
		Avatar:         user.Avatar,
		Email:          user.Email,
		Phone:          user.Phone,
		Quota:          user.Quota,
		UsedQuota:      user.UsedQuota,
		QuotaExpiresAt: user.QuotaExpiresAt,
		Models:         user.Models,
		Status:         user.Status,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的用户信息
func (s *sUser) RemoveCacheUser(ctx context.Context, userId int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser RemoveCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.userCache.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := s.userQuotaCache.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存用户额度到缓存
func (s *sUser) SaveCacheUserQuota(ctx context.Context, userId, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser SaveCacheUserQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.userQuotaCache.Set(ctx, userId, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的用户额度
func (s *sUser) GetCacheUserQuota(ctx context.Context, userId int) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser GetCacheUserQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if userQuotaValue := s.userQuotaCache.GetVal(ctx, userId); userQuotaValue != nil {
		return userQuotaValue.(int)
	}

	return 0
}

// 变更订阅
func (s *sUser) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sUser Subscribe: %s", gjson.MustEncodeString(message))

	var user *entity.User
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS, consts.ACTION_MODELS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &user); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheUser(ctx, user)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &user); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheUser(ctx, user.UserId)

	case consts.ACTION_CACHE:

		var userQuota *model.UserQuota
		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &userQuota); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := s.SaveCacheUserQuota(ctx, userQuota.UserId, userQuota.CurrentQuota); err != nil {
			logger.Error(ctx, err)
		}
	}

	return nil
}
