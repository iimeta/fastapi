package dao

import (
	"context"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
	"go.mongodb.org/mongo-driver/bson"
)

var Robot = NewRobotDao()

type RobotDao struct {
	*MongoDB[entity.Robot]
}

func NewRobotDao(database ...string) *RobotDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &RobotDao{
		MongoDB: NewMongoDB[entity.Robot](database[0], do.ROBOT_COLLECTION),
	}
}

// 获取登录机器的信息
func (d *RobotDao) GetLoginRobot(ctx context.Context) (*entity.Robot, error) {

	robot, err := d.FindOne(ctx, bson.M{"user_id": 1, "status": consts.RootStatusNormal})
	if err != nil {
		return nil, err
	}

	return robot, nil
}

// 根据绑定userId获取机器人的信息
func (d *RobotDao) GetRobotByUserId(ctx context.Context, userId int) (*entity.Robot, error) {

	robot, err := d.FindOne(ctx, bson.M{"user_id": userId, "status": consts.RootStatusNormal})
	if err != nil {
		return nil, err
	}

	return robot, nil
}

// 获取机器人列表
func (d *RobotDao) GetRobotList(ctx context.Context, userIds ...int) ([]*entity.Robot, error) {

	filter := bson.M{
		"is_talk": 1,
		"status":  consts.RootStatusNormal,
	}

	if len(userIds) > 0 {
		filter["user_id"] = bson.M{
			"$in": userIds,
		}
	}

	robotList, err := d.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return robotList, nil
}

// 获取机器人用户列表
func (d *RobotDao) GetRobotUserList(ctx context.Context) ([]*entity.User, error) {

	robotList, err := d.GetRobotList(ctx)
	if err != nil {
		return nil, err
	}

	if len(robotList) == 0 {
		return nil, err
	}

	robotUserIds := make([]int, 0)
	for _, robot := range robotList {
		if robot.UserId == 1 {
			continue
		}
		robotUserIds = append(robotUserIds, robot.UserId)
	}

	userList, err := User.FindUserListByUserIds(ctx, robotUserIds)
	if err != nil {
		return nil, err
	}

	return userList, nil
}
