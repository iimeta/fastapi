package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var Image = NewImageDao()

type ImageDao struct {
	*MongoDB[entity.Image]
}

func NewImageDao(database ...string) *ImageDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ImageDao{
		MongoDB: NewMongoDB[entity.Image](database[0], IMAGE),
	}
}
