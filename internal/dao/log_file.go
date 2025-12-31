package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var LogFile = NewLogFileDao()

type LogFileDao struct {
	*MongoDB[entity.LogFile]
}

func NewLogFileDao(database ...string) *LogFileDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogFileDao{
		MongoDB: NewMongoDB[entity.LogFile](database[0], LOG_FILE),
	}
}
