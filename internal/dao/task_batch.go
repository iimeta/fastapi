package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var TaskBatch = NewTaskBatchDao()

type TaskBatchDao struct {
	*MongoDB[entity.TaskBatch]
}

func NewTaskBatchDao(database ...string) *TaskBatchDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &TaskBatchDao{
		MongoDB: NewMongoDB[entity.TaskBatch](database[0], TASK_BATCH),
	}
}
