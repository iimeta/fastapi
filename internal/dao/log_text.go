package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var LogText = NewLogTextDao()

type LogTextDao struct {
	*MongoDB[entity.LogText]
}

func NewLogTextDao(database ...string) *LogTextDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogTextDao{
		MongoDB: NewMongoDB[entity.LogText](database[0], LOG_TEXT),
	}
}
