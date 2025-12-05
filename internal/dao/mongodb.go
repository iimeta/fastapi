package dao

import (
	"context"
	"reflect"

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
)

type IMongoDB any

type MongoDB[T IMongoDB] struct {
	*db.MongoDB
}

type FindOptions struct {
	SortFields    []string // 排序字段
	Index         string   // 查询索引
	IncludeFields []string // 包含字段
	ExcludeFields []string // 排除字段
}

func NewMongoDB[T IMongoDB](database, collection string) *MongoDB[T] {
	return &MongoDB[T]{
		MongoDB: &db.MongoDB{
			Database:   database,
			Collection: collection,
		}}
}

func (m *MongoDB[T]) Find(ctx context.Context, filter map[string]any, findOptions ...*FindOptions) ([]*T, error) {

	var result []*T
	if err := find(ctx, m.Database, m.Collection, filter, &result, findOptions...); err != nil {
		return nil, err
	}

	return result, nil
}

func find(ctx context.Context, database, collection string, filter map[string]any, result any, findOptions ...*FindOptions) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if len(findOptions) > 0 {
		m.SortFields = findOptions[0].SortFields
		m.Index = findOptions[0].Index
		m.IncludeFields = findOptions[0].IncludeFields
		m.ExcludeFields = findOptions[0].ExcludeFields
	}

	return m.Find(ctx, result)
}

func (m *MongoDB[T]) FindOne(ctx context.Context, filter map[string]any, findOptions ...*FindOptions) (*T, error) {

	var result *T
	if err := findOne(ctx, m.Database, m.Collection, filter, &result, findOptions...); err != nil {
		return nil, err
	}

	return result, nil
}

func findOne(ctx context.Context, database, collection string, filter map[string]any, result any, findOptions ...*FindOptions) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if len(findOptions) > 0 {
		m.SortFields = findOptions[0].SortFields
		m.Index = findOptions[0].Index
		m.IncludeFields = findOptions[0].IncludeFields
		m.ExcludeFields = findOptions[0].ExcludeFields
	}

	return m.FindOne(ctx, result)
}

func (m *MongoDB[T]) FindById(ctx context.Context, id any, findOptions ...*FindOptions) (*T, error) {

	var result *T
	if err := findById(ctx, m.Database, m.Collection, id, &result, findOptions...); err != nil {
		return nil, err
	}

	return result, nil
}

func findById(ctx context.Context, database, collection string, id, result any, findOptions ...*FindOptions) error {

	filter := bson.M{"_id": id}

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if len(findOptions) > 0 {
		m.SortFields = findOptions[0].SortFields
		m.Index = findOptions[0].Index
		m.IncludeFields = findOptions[0].IncludeFields
		m.ExcludeFields = findOptions[0].ExcludeFields
	}

	return m.FindOne(ctx, result)
}

func (m *MongoDB[T]) FindByIds(ctx context.Context, ids any, findOptions ...*FindOptions) ([]*T, error) {

	var result []*T
	if err := findByIds(ctx, m.Database, m.Collection, ids, &result, findOptions...); err != nil {
		return nil, err
	}

	return result, nil
}

func findByIds(ctx context.Context, database, collection string, ids, result any, findOptions ...*FindOptions) error {

	filter := bson.M{"_id": bson.M{"$in": ids}}

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if len(findOptions) > 0 {
		m.SortFields = findOptions[0].SortFields
		m.Index = findOptions[0].Index
		m.IncludeFields = findOptions[0].IncludeFields
		m.ExcludeFields = findOptions[0].ExcludeFields
	}

	return m.Find(ctx, result)
}

func (m *MongoDB[T]) FindByPage(ctx context.Context, paging *db.Paging, filter map[string]any, findOptions ...*FindOptions) ([]*T, error) {

	var result []*T

	if err := findByPage(ctx, m.Database, m.Collection, paging, filter, &result, findOptions...); err != nil {
		return nil, err
	}

	return result, nil
}

func findByPage(ctx context.Context, database, collection string, paging *db.Paging, filter map[string]any, result any, findOptions ...*FindOptions) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	if len(findOptions) > 0 {
		m.SortFields = findOptions[0].SortFields
		m.Index = findOptions[0].Index
		m.IncludeFields = findOptions[0].IncludeFields
		m.ExcludeFields = findOptions[0].ExcludeFields
	}

	return m.FindByPage(ctx, paging, result)
}

func (m *MongoDB[T]) Insert(ctx context.Context, document any) (string, error) {
	return insert(ctx, m.Database, document)
}

func insert(ctx context.Context, database string, document any) (string, error) {

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
	if value["_id"] == nil || value["_id"] == "" {
		value["_id"] = util.GenerateId()
	}

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

func (m *MongoDB[T]) Inserts(ctx context.Context, documents []any) ([]string, error) {
	return inserts(ctx, m.Database, documents)
}

func inserts(ctx context.Context, database string, documents []any) ([]string, error) {

	collection := gmeta.Get(documents[0], "collection").String()
	if collection == "" {
		return nil, errors.New("collection meta undefined")
	}

	values := make([]any, 0)
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
		if value["_id"] == nil || value["_id"] == "" {
			value["_id"] = util.GenerateId()
		}

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

func (m *MongoDB[T]) UpdateById(ctx context.Context, id, update any, isUpsert ...bool) error {
	return m.UpdateOne(ctx, bson.M{"_id": id}, update, isUpsert...)
}

func (m *MongoDB[T]) UpdateOne(ctx context.Context, filter map[string]any, update any, isUpsert ...bool) error {
	return updateOne(ctx, m.Database, m.Collection, filter, update, isUpsert...)
}

func updateOne(ctx context.Context, database, collection string, filter map[string]any, update any, isUpsert ...bool) error {

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
		for k, v := range value {

			if gstr.Contains(k, "$") {
				continue
			}

			if value["$set"] != nil {
				setValues := gconv.Map(value["$set"])
				setValues[k] = v
				value["$set"] = setValues
			} else {
				value["$set"] = bson.M{
					k: v,
				}
			}

			delete(value, k)
		}

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

		update = value
	}

	opt := &options.UpdateOptions{}
	if len(isUpsert) > 0 && isUpsert[0] {
		opt.SetUpsert(true)
	}

	return m.UpdateOne(ctx, update, opt)
}

func (m *MongoDB[T]) UpdateMany(ctx context.Context, filter map[string]any, update any, isUpsert ...bool) error {
	return updateMany(ctx, m.Database, m.Collection, filter, update, isUpsert...)
}

func updateMany(ctx context.Context, database, collection string, filter map[string]any, update any, isUpsert ...bool) error {

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
		for k, v := range value {

			if gstr.Contains(k, "$") {
				continue
			}

			if value["$set"] != nil {
				setValues := gconv.Map(value["$set"])
				setValues[k] = v
				value["$set"] = setValues
			} else {
				value["$set"] = bson.M{
					k: v,
				}
			}

			delete(value, k)
		}

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

		update = value
	}

	opt := &options.UpdateOptions{}
	if len(isUpsert) > 0 && isUpsert[0] {
		opt.SetUpsert(true)
	}

	return m.UpdateMany(ctx, update, opt)
}

func (m *MongoDB[T]) FindOneAndUpdateById(ctx context.Context, id any, update any, isUpsert ...bool) (*T, error) {
	return m.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, isUpsert...)
}

func (m *MongoDB[T]) FindOneAndUpdate(ctx context.Context, filter map[string]any, update any, isUpsert ...bool) (*T, error) {

	var result *T
	if err := findOneAndUpdate(ctx, m.Database, m.Collection, filter, update, &result, isUpsert...); err != nil {
		return nil, err
	}

	return result, nil
}

func findOneAndUpdate(ctx context.Context, database, collection string, filter map[string]any, update any, result any, isUpsert ...bool) error {

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
		for k, v := range value {

			if gstr.Contains(k, "$") {
				continue
			}

			if value["$set"] != nil {
				setValues := gconv.Map(value["$set"])
				setValues[k] = v
				value["$set"] = setValues
			} else {
				value["$set"] = bson.M{
					k: v,
				}
			}

			delete(value, k)
		}

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

		update = value
	}

	opt := &options.FindOneAndUpdateOptions{}
	opt.SetReturnDocument(options.After)

	if len(isUpsert) > 0 && isUpsert[0] {
		opt.SetUpsert(true)
	}

	return m.FindOneAndUpdate(ctx, update, result, opt)
}

func (m *MongoDB[T]) DeleteById(ctx context.Context, id any) (int64, error) {
	return m.DeleteOne(ctx, bson.M{"_id": id})
}

func (m *MongoDB[T]) DeleteOne(ctx context.Context, filter map[string]any) (int64, error) {
	return deleteOne(ctx, m.Database, m.Collection, filter)
}

func deleteOne(ctx context.Context, database, collection string, filter map[string]any) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.DeleteOne(ctx)
}

func (m *MongoDB[T]) DeleteMany(ctx context.Context, filter map[string]any) (int64, error) {
	return deleteMany(ctx, m.Database, m.Collection, filter)
}

func deleteMany(ctx context.Context, database, collection string, filter map[string]any) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.DeleteMany(ctx)
}

func (m *MongoDB[T]) FindOneAndDeleteById(ctx context.Context, id any) (*T, error) {
	return m.FindOneAndDelete(ctx, bson.M{"_id": id})
}

func (m *MongoDB[T]) FindOneAndDelete(ctx context.Context, filter map[string]any) (*T, error) {

	var result *T
	if err := findOneAndDelete(ctx, m.Database, m.Collection, filter, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func findOneAndDelete(ctx context.Context, database, collection string, filter map[string]any, result any) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.FindOneAndDelete(ctx, result)
}

func (m *MongoDB[T]) CountDocuments(ctx context.Context, filter map[string]any) (int64, error) {
	return countDocuments(ctx, m.Database, m.Collection, filter)
}

func countDocuments(ctx context.Context, database, collection string, filter map[string]any) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Filter:     filter,
	}

	return m.CountDocuments(ctx)
}

func (m *MongoDB[T]) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	return estimatedDocumentCount(ctx, m.Database, m.Collection)
}

func estimatedDocumentCount(ctx context.Context, database, collection string) (int64, error) {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
	}

	return m.EstimatedDocumentCount(ctx)
}

func (m *MongoDB[T]) Aggregate(ctx context.Context, pipeline []bson.M, result any) error {
	return Aggregate(ctx, m.Database, m.Collection, pipeline, result)
}

func Aggregate(ctx context.Context, database, collection string, pipeline []bson.M, result any) error {

	m := &db.MongoDB{
		Database:   database,
		Collection: collection,
		Pipeline:   pipeline,
	}

	return m.Aggregate(ctx, result)
}

func (m *MongoDB[T]) AggregateByPage(ctx context.Context, countField string, paging *db.Paging, countPipeline, pipeline []bson.M, result any) error {
	return AggregateByPage(ctx, m.Database, m.Collection, countField, paging, countPipeline, pipeline, result)
}

func AggregateByPage(ctx context.Context, database, collection, countField string, paging *db.Paging, countPipeline, pipeline []bson.M, result any) error {

	m := &db.MongoDB{
		Database:      database,
		Collection:    collection,
		Pipeline:      pipeline,
		CountPipeline: countPipeline,
	}

	countResult := make([]map[string]any, 0)
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
func isStruct(value any) bool {

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
