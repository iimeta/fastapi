package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var Group = NewGroupDao()

type GroupDao struct {
	*MongoDB[entity.Group]
}

func NewGroupDao(database ...string) *GroupDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &GroupDao{
		MongoDB: NewMongoDB[entity.Group](database[0], GROUP),
	}
}
