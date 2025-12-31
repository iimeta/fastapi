package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var LogMidjourney = NewLogMidjourneyDao()

type LogMidjourneyDao struct {
	*MongoDB[entity.LogMidjourney]
}

func NewLogMidjourneyDao(database ...string) *LogMidjourneyDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogMidjourneyDao{
		MongoDB: NewMongoDB[entity.LogMidjourney](database[0], LOG_MIDJOURNEY),
	}
}
