package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var ModelAgent = NewModelAgentDao()

type ModelAgentDao struct {
	*MongoDB[entity.ModelAgent]
}

func NewModelAgentDao(database ...string) *ModelAgentDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ModelAgentDao{
		MongoDB: NewMongoDB[entity.ModelAgent](database[0], MODEL_AGENT),
	}
}
