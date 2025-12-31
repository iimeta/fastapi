package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var LogVideo = NewLogVideoDao()

type LogVideoDao struct {
	*MongoDB[entity.LogVideo]
}

func NewLogVideoDao(database ...string) *LogVideoDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogVideoDao{
		MongoDB: NewMongoDB[entity.LogVideo](database[0], LOG_VIDEO),
	}
}
