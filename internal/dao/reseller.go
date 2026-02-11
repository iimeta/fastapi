package dao

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var Reseller = NewResellerDao()

type ResellerDao struct {
	*MongoDB[entity.Reseller]
}

func NewResellerDao(database ...string) *ResellerDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ResellerDao{
		MongoDB: NewMongoDB[entity.Reseller](database[0], RESELLER),
	}
}

// 根据userId查询用户
func (d *ResellerDao) FindResellerByUserId(ctx context.Context, userId int) (*entity.Reseller, error) {
	return d.FindOne(ctx, bson.M{"user_id": userId, "status": 1})
}
