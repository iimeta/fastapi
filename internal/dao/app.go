package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var App = NewAppDao()

type AppDao struct {
	*MongoDB[entity.App]
}

func NewAppDao(database ...string) *AppDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &AppDao{
		MongoDB: NewMongoDB[entity.App](database[0], APP),
	}
}
