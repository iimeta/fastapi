package user

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sUser struct{}

func init() {
	service.RegisterUser(New())
}

func New() service.IUser {
	return &sUser{}
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
		Gender: user.Gender,
		Email:  user.Email,
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
			Gender: result.Gender,
			Email:  result.Email,
			Quota:  result.Quota,
			Remark: result.Remark,
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

// 变更订阅
func (s *sUser) Subscribe(ctx context.Context, msg string) error {

	user := new(entity.User)
	err := gjson.Unmarshal([]byte(msg), &user)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	fmt.Println(gjson.MustEncodeString(user))

	return nil
}
