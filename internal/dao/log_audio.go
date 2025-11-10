package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var LogAudio = NewLogAudioDao()

type LogAudioDao struct {
	*MongoDB[entity.LogAudio]
}

func NewLogAudioDao(database ...string) *LogAudioDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &LogAudioDao{
		MongoDB: NewMongoDB[entity.LogAudio](database[0], LOG_AUDIO),
	}
}
