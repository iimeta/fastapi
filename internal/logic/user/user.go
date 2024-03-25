package user

import (
	"context"
	"fmt"
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
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
)

type sUser struct {
	userCache *cache.Cache // [userId]User
}

func init() {
	service.RegisterUser(New())
}

func New() service.IUser {
	return &sUser{
		userCache: cache.New(),
	}
}

// 根据userId获取用户信息
func (s *sUser) GetUserByUserId(ctx context.Context, userId int) (*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetUserByUserId time: %d", gtime.TimestampMilli()-now)
	}()

	user, err := dao.User.FindUserByUserId(ctx, userId)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.User{
		Id:     user.Id,
		UserId: user.UserId,
		Name:   user.Name,
		Avatar: user.Avatar,
		Email:  user.Email,
		Phone:  user.Phone,
		Quota:  user.Quota,
		Models: user.Models,
		Status: user.Status,
	}, nil
}

// 用户列表
func (s *sUser) List(ctx context.Context) ([]*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{
		"status": 1,
	}

	results, err := dao.User.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.User, 0)
	for _, result := range results {
		items = append(items, &model.User{
			Id:     result.Id,
			UserId: result.UserId,
			Name:   result.Name,
			Avatar: result.Avatar,
			Email:  result.Email,
			Phone:  result.Phone,
			Quota:  result.Quota,
			Models: result.Models,
			Status: result.Status,
		})
	}

	return items, nil
}

// 更改用户额度
func (s *sUser) ChangeQuota(ctx context.Context, userId, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sUser ChangeQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.User.UpdateOne(ctx, bson.M{"user_id": userId}, bson.M{
		"$inc": bson.M{
			"quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存用户信息到缓存
func (s *sUser) SaveCacheUser(ctx context.Context, user *model.User) error {

	if user == nil {
		return errors.New("user is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_USER_KEY, user.UserId), user); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveUser(ctx, user)

	if err := s.userCache.Set(ctx, user.UserId, user, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的用户信息
func (s *sUser) GetCacheUser(ctx context.Context, userId int) (*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if user := service.Session().GetUser(ctx); user != nil {
		return user, nil
	}

	if userCacheValue := s.userCache.GetVal(ctx, userId); userCacheValue != nil {
		return userCacheValue.(*model.User), nil
	}

	reply, err := redis.Get(ctx, fmt.Sprintf(consts.API_USER_KEY, userId))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || reply.IsNil() {
		return nil, errors.New("user is nil")
	}

	user := new(model.User)
	if err = reply.Struct(&user); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	service.Session().SaveUser(ctx, user)

	if err = s.userCache.Set(ctx, user.UserId, user, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return user, nil
}

// 更新缓存中的用户信息
func (s *sUser) UpdateCacheUser(ctx context.Context, user *entity.User) {
	if err := s.SaveCacheUser(ctx, &model.User{
		Id:     user.Id,
		UserId: user.UserId,
		Name:   user.Name,
		Avatar: user.Avatar,
		Email:  user.Email,
		Phone:  user.Phone,
		Quota:  user.Quota,
		Models: user.Models,
		Status: user.Status,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的用户信息
func (s *sUser) RemoveCacheUser(ctx context.Context, userId int) {

	if _, err := s.userCache.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_USER_KEY, userId)); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sUser) Subscribe(ctx context.Context, msg string) error {

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sUser Subscribe: %s", gjson.MustEncodeString(message))

	var user *entity.User
	switch message.Action {
	case consts.ACTION_UPDATE, consts.ACTION_STATUS:

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
	}

	return nil
}
