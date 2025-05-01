package dao

import (
	"context"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
	"go.mongodb.org/mongo-driver/bson"
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
		MongoDB: NewMongoDB[entity.Reseller](database[0], do.RESELLER_COLLECTION),
	}
}

// 根据userId查询用户
func (d *ResellerDao) FindResellerByUserId(ctx context.Context, userId int) (*entity.Reseller, error) {
	return d.FindOne(ctx, bson.M{"user_id": userId, "status": 1})
}
