package dao

import (
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
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
		MongoDB: NewMongoDB[entity.Group](database[0], do.GROUP_COLLECTION),
	}
}
