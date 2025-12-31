package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var TaskVideo = NewTaskVideoDao()

type TaskVideoDao struct {
	*MongoDB[entity.TaskVideo]
}

func NewTaskVideoDao(database ...string) *TaskVideoDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &TaskVideoDao{
		MongoDB: NewMongoDB[entity.TaskVideo](database[0], TASK_VIDEO),
	}
}
