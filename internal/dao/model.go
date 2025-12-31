package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var Model = NewModelDao()

type ModelDao struct {
	*MongoDB[entity.Model]
}

func NewModelDao(database ...string) *ModelDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ModelDao{
		MongoDB: NewMongoDB[entity.Model](database[0], MODEL),
	}
}
