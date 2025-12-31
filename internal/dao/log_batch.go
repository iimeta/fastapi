package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var LogBatch = NewLogBatchDao()

type LogBatchDao struct {
	*MongoDB[entity.LogBatch]
}

func NewLogBatchDao(database ...string) *LogBatchDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogBatchDao{
		MongoDB: NewMongoDB[entity.LogBatch](database[0], LOG_BATCH),
	}
}
