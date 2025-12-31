package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var TaskFile = NewTaskFileDao()

type TaskFileDao struct {
	*MongoDB[entity.TaskFile]
}

func NewTaskFileDao(database ...string) *TaskFileDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &TaskFileDao{
		MongoDB: NewMongoDB[entity.TaskFile](database[0], TASK_FILE),
	}
}
