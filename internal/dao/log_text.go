package dao

import (
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/db"
)

var Chat = NewChatDao()

type ChatDao struct {
	*MongoDB[entity.Chat]
}

func NewChatDao(database ...string) *ChatDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ChatDao{
		MongoDB: NewMongoDB[entity.Chat](database[0], CHAT),
	}
}
