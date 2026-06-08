package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var TaskImage = NewTaskImageDao()

type TaskImageDao struct {
	*MongoDB[entity.TaskImage]
}

func NewTaskImageDao(database ...string) *TaskImageDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &TaskImageDao{
		MongoDB: NewMongoDB[entity.TaskImage](database[0], TASK_IMAGE),
	}
}
