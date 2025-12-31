package dao

import (
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/utility/db"
)

var Provider = NewProviderDao()

type ProviderDao struct {
	*MongoDB[entity.Provider]
}

func NewProviderDao(database ...string) *ProviderDao {

	if len(database) == 0 {
		database = append(database, db.DefaultDatabase)
	}

	return &ProviderDao{
		MongoDB: NewMongoDB[entity.Provider](database[0], PROVIDER),
	}
}
