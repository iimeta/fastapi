package user

import (
	"context"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
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
func (s *sUser) GetUserByUid(ctx context.Context, userId int) (*model.User, error) {

	user, err := dao.User.FindUserByUserId(ctx, userId)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.User{
		Id:        user.Id,
		UserId:    user.UserId,
		Mobile:    user.Mobile,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// 用户列表
func (s *sUser) List(ctx context.Context) ([]*model.User, error) {

	filter := bson.M{}

	results, err := dao.User.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.User, 0)
	for _, result := range results {
		items = append(items, &model.User{
			Id:       result.Id,
			UserId:   result.UserId,
			Nickname: result.Nickname,
			Avatar:   result.Avatar,
			Gender:   result.Gender,
			Mobile:   result.Mobile,
			Email:    result.Email,
			Quota:    result.Quota,
			Remark:   result.Remark,
		})
	}

	return items, nil
}
