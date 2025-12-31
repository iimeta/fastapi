package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var Key = NewKeyDao()

type KeyDao struct {
	*MongoDB[entity.Key]
}

func NewKeyDao(database ...string) *KeyDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &KeyDao{
		MongoDB: NewMongoDB[entity.Key](database[0], KEY),
	}
}
