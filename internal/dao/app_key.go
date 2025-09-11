package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var AppKey = NewAppKeyDao()

type AppKeyDao struct {
	*MongoDB[entity.AppKey]
}

func NewAppKeyDao(database ...string) *AppKeyDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &AppKeyDao{
		MongoDB: NewMongoDB[entity.AppKey](database[0], APP_KEY),
	}
}
