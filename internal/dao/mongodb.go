package dao

import (
	"context"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/gmeta"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/db"
	"github.com/iimeta/fastapi/utility/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
)

type IMongoDB interface{}

type MongoDB[T IMongoDB] struct {
	*db.MongoDB
}

func NewMongoDB[T IMongoDB](database, collection string) *MongoDB[T] {
	return &MongoDB[T]{
		MongoDB: &db.MongoDB{
			Database:   database,
			Collection: collection,
		}}
}

func (m *MongoDB[T]) Find(ctx context.Context, filter map[string]interface{}, sortFields ...string) ([]*T, error) {

	var result []*T
	if err := Find(ctx, m.Database, m.Collection, filter, &result, sortFields...); err != nil {
		return nil, err
	}

	return result, nil
}

func Find(ctx context.Context, database, collection string, filter map[string]interface{}, result interface{}, sortFields ...string) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.Find(ctx, result, sortFields...)
}

func (m *MongoDB[T]) FindOne(ctx context.Context, filter map[string]interface{}, sortFields ...string) (*T, error) {

	var result *T
	if err := FindOne(ctx, m.Database, m.Collection, filter, &result, sortFields...); err != nil {
		return nil, err
	}

	return result, nil
}

func FindOne(ctx context.Context, database, collection string, filter map[string]interface{}, result interface{}, sortFields ...string) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.FindOne(ctx, result, sortFields...)
}

func (m *MongoDB[T]) FindById(ctx context.Context, id interface{}) (*T, error) {

	var result *T
	if err := FindById(ctx, m.Database, m.Collection, id, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func FindById(ctx context.Context, database, collection string, id, result interface{}) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     bson.M{"_id": id},
	}

	return m.FindOne(ctx, result)
}

func (m *MongoDB[T]) FindByIds(ctx context.Context, ids interface{}) ([]*T, error) {

	var result []*T
	if err := FindByIds(ctx, m.Database, m.Collection, ids, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func FindByIds(ctx context.Context, database, collection string, ids, result interface{}) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     bson.M{"_id": bson.M{"$in": ids}},
	}

	return m.Find(ctx, result)
}

func (m *MongoDB[T]) FindByPage(ctx context.Context, paging *db.Paging, filter map[string]interface{}, sortFields ...string) ([]*T, error) {

	var result []*T
	if err := FindByPage(ctx, m.Database, m.Collection, paging, filter, &result, sortFields...); err != nil {
		return nil, err
	}

	return result, nil
}

func FindByPage(ctx context.Context, database, collection string, paging *db.Paging, filter map[string]interface{}, result interface{}, sortFields ...string) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.FindByPage(ctx, paging, result, sortFields...)
}

func (m *MongoDB[T]) Insert(ctx context.Context, document interface{}) (string, error) {
	return Insert(ctx, m.Database, document)
}

func Insert(ctx context.Context, database string, document interface{}) (string, error) {

	collection := gmeta.Get(document, "collection").String()
	if collection == "" {
		return "", errors.New("collection meta undefined")
	}

	bytes, err := bson.Marshal(document)
	if err != nil {
		return "", err
	}

	value := bson.M{}
	if err = bson.Unmarshal(bytes, &value); err != nil {
		return "", err
	}

	// 统一主键成int类型的string格式, 雪花ID
	value["_id"] = util.GenerateId()

	if value["creator"] == nil || value["creator"] == "" {
		value["creator"] = service.Session().GetSecretKey(ctx)
	}

	if value["created_at"] == nil || gconv.Int(value["created_at"]) == 0 {
		value["created_at"] = gtime.TimestampMilli()
	}

	if value["updated_at"] == nil || gconv.Int(value["updated_at"]) == 0 {
		value["updated_at"] = gtime.TimestampMilli()
	}

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
	}

	id, err := m.InsertOne(ctx, value)
	if err != nil {
		return "", err
	}

	return gconv.String(id), nil
}

func (m *MongoDB[T]) Inserts(ctx context.Context, documents []interface{}) ([]string, error) {
	return Inserts(ctx, m.Database, documents)
}

func Inserts(ctx context.Context, database string, documents []interface{}) ([]string, error) {

	collection := gmeta.Get(documents[0], "collection").String()
	if collection == "" {
		return nil, errors.New("collection meta undefined")
	}

	values := make([]interface{}, 0)
	for _, document := range documents {

		bytes, err := bson.Marshal(document)
		if err != nil {
			return nil, err
		}

		value := bson.M{}
		if err = bson.Unmarshal(bytes, &value); err != nil {
			return nil, err
		}

		// 统一主键成int类型的string格式, 雪花ID
		value["_id"] = util.GenerateId()

		if value["creator"] == nil || value["creator"] == "" {
			value["creator"] = service.Session().GetSecretKey(ctx)
		}

		if value["created_at"] == nil || gconv.Int(value["created_at"]) == 0 {
			value["created_at"] = gtime.TimestampMilli()
		}

		if value["updated_at"] == nil || gconv.Int(value["updated_at"]) == 0 {
			value["updated_at"] = gtime.TimestampMilli()
		}

		values = append(values, value)
	}

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
	}

	ids, err := m.InsertMany(ctx, values)
	if err != nil {
		return nil, err
	}

	return gconv.Strings(ids), nil
}

func (m *MongoDB[T]) UpdateById(ctx context.Context, id, update interface{}, isUpsert ...bool) error {
	return UpdateById(ctx, m.Database, m.Collection, id, update, isUpsert...)
}

func UpdateById(ctx context.Context, database, collection string, id, update interface{}, isUpsert ...bool) error {
	return UpdateOne(ctx, database, collection, bson.M{"_id": id}, update, isUpsert...)
}

func (m *MongoDB[T]) UpdateOne(ctx context.Context, filter map[string]interface{}, update interface{}, isUpsert ...bool) error {
	return UpdateOne(ctx, m.Database, m.Collection, filter, update, isUpsert...)
}

func UpdateOne(ctx context.Context, database, collection string, filter map[string]interface{}, update interface{}, isUpsert ...bool) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if isStruct(update) {

		bytes, err := bson.Marshal(update)
		if err != nil {
			return err
		}

		value := bson.M{}
		if err = bson.Unmarshal(bytes, &value); err != nil {
			return err
		}

		if value["updater"] == nil || value["updater"] == "" {
			value["updater"] = service.Session().GetSecretKey(ctx)
		}

		if value["updated_at"] == nil || gconv.Int(value["updated_at"]) == 0 {
			value["updated_at"] = gtime.TimestampMilli()
		}

		update = bson.M{
			"$set": value,
		}

	} else {

		value := gconv.Map(update)

		containKey := false
		for key := range value {
			if gstr.Contains(key, "$") {
				containKey = true
				break
			}
		}

		if containKey {

			if value["updater"] == nil || value["updater"] == "" {
				if value["$set"] != nil {
					setValues := gconv.Map(value["$set"])
					if setValues["updater"] == nil || setValues["updater"] == "" {
						setValues["updater"] = service.Session().GetSecretKey(ctx)
						value["$set"] = setValues
					}
				} else {
					value["$set"] = bson.M{
						"updater": service.Session().GetSecretKey(ctx),
					}
				}
			}

			if value["updated_at"] == nil || gconv.Int(value["updated_at"]) == 0 {
				if value["$set"] != nil {
					setValues := gconv.Map(value["$set"])
					if setValues["updated_at"] == nil || gconv.Int(setValues["updated_at"]) == 0 {
						setValues["updated_at"] = gtime.TimestampMilli()
						value["$set"] = setValues
					}
				} else {
					value["$set"] = bson.M{
						"updated_at": gtime.TimestampMilli(),
					}
				}
			}
		} else {

			if value["updater"] == nil || value["updater"] == "" {
				value["updater"] = service.Session().GetSecretKey(ctx)
			}

			if value["updated_at"] == nil || gconv.Int(value["updated_at"]) == 0 {
				value["updated_at"] = gtime.TimestampMilli()
			}
		}

		if !containKey {
			update = bson.M{
				"$set": value,
			}
		} else {
			update = value
		}
	}

	opt := &options.UpdateOptions{}
	if len(isUpsert) > 0 && isUpsert[0] {
		opt.SetUpsert(true)
	}

	return m.UpdateOne(ctx, update, opt)
}

func (m *MongoDB[T]) UpdateMany(ctx context.Context, filter map[string]interface{}, update interface{}, isUpsert ...bool) error {
	return UpdateMany(ctx, m.Database, m.Collection, filter, update, isUpsert...)
}

func UpdateMany(ctx context.Context, database, collection string, filter map[string]interface{}, update interface{}, isUpsert ...bool) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if isStruct(update) {

		bytes, err := bson.Marshal(update)
		if err != nil {
			return err
		}

		value := bson.M{}
		if err = bson.Unmarshal(bytes, &value); err != nil {
			return err
		}

		update = bson.M{
			"$set": value,
		}

	} else {

		containKey := false
		for key := range gconv.Map(update) {
			if gstr.Contains(key, "$") {
				containKey = true
				break
			}
		}

		if !containKey {
			update = bson.M{
				"$set": update,
			}
		}
	}

	opt := &options.UpdateOptions{}
	if len(isUpsert) > 0 && isUpsert[0] {
		opt.SetUpsert(true)
	}

	return m.UpdateMany(ctx, update, opt)
}

func (m *MongoDB[T]) DeleteById(ctx context.Context, id interface{}) error {
	return DeleteById(ctx, m.Database, m.Collection, id)
}

func DeleteById(ctx context.Context, database, collection string, id interface{}) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
	}

	return m.DeleteById(ctx, id)
}

func (m *MongoDB[T]) DeleteOne(ctx context.Context, filter map[string]interface{}) (int64, error) {
	return DeleteOne(ctx, m.Database, m.Collection, filter)
}

func DeleteOne(ctx context.Context, database, collection string, filter map[string]interface{}) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.DeleteOne(ctx)
}

func (m *MongoDB[T]) DeleteMany(ctx context.Context, filter map[string]interface{}) (int64, error) {
	return DeleteMany(ctx, m.Database, m.Collection, filter)
}

func DeleteMany(ctx context.Context, database, collection string, filter map[string]interface{}) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.DeleteMany(ctx)
}

func (m *MongoDB[T]) CountDocuments(ctx context.Context, filter map[string]interface{}) (int64, error) {
	return CountDocuments(ctx, m.Database, m.Collection, filter)
}

func CountDocuments(ctx context.Context, database, collection string, filter map[string]interface{}) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.CountDocuments(ctx)
}

func (m *MongoDB[T]) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	return EstimatedDocumentCount(ctx, m.Database, m.Collection)
}

func EstimatedDocumentCount(ctx context.Context, database, collection string) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
	}

	return m.EstimatedDocumentCount(ctx)
}

func (m *MongoDB[T]) Aggregate(ctx context.Context, pipeline []bson.M, result interface{}) error {
	return Aggregate(ctx, m.Database, m.Collection, pipeline, result)
}

func Aggregate(ctx context.Context, database, collection string, pipeline []bson.M, result interface{}) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Pipeline:   pipeline,
	}

	return m.Aggregate(ctx, result)
}

func (m *MongoDB[T]) AggregateByPage(ctx context.Context, countField string, paging *db.Paging, countPipeline, pipeline []bson.M, result interface{}) error {
	return AggregateByPage(ctx, m.Database, m.Collection, countField, paging, countPipeline, pipeline, result)
}

func AggregateByPage(ctx context.Context, database, collection, countField string, paging *db.Paging, countPipeline, pipeline []bson.M, result interface{}) error {

	m := &db.MongoDB{
		Database:      database,
		Collection:    collection,
		Pipeline:      pipeline,
		CountPipeline: countPipeline,
	}

	countResult := make([]map[string]interface{}, 0)
	if err := m.AggregateByPage(ctx, &countResult, result); err != nil {
		return err
	}

	if len(countResult) > 0 {
		paging.Total = int64(countResult[0][countField].(int32))
		paging.GetPages()
	}

	return nil
}

// 判断底层类型是否为Struct
func isStruct(value interface{}) bool {

	// 获取值的类型
	valueType := reflect.TypeOf(value)

	kind := valueType.Kind()

	if kind == reflect.Struct {
		return true
	} else if kind == reflect.Ptr { // 判断是否为指针类型

		// 获取指针指向的值的类型
		elemType := valueType.Elem()

		// 判断指针指向的值的类型是否为struct
		if elemType.Kind() == reflect.Struct {
			return true
		}
	}

	return false
}
