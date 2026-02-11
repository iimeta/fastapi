package db

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDB struct {
	Database      string
	Collection    string
	Filter        map[string]any
	SortFields    []string // 排序字段
	Index         string   // 查询索引
	IncludeFields []string // 包含字段
	ExcludeFields []string // 排除字段
	CountPipeline []bson.M // AggregateByPage
	Pipeline      []bson.M // Aggregate/AggregateByPage
}

func (m *MongoDB) Find(ctx context.Context, result any) error {

	var findOptions []options.Lister[options.FindOptions]

	findOptions = append(findOptions, options.Find().SetAllowDiskUse(true))

	if len(m.SortFields) > 0 {

		sort := bson.D{}
		for _, field := range m.SortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOptions = append(findOptions, options.Find().SetSort(sort))
	}

	if m.Index != "" {
		findOptions = append(findOptions, options.Find().SetHint(m.Index))
	}

	if len(m.IncludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.IncludeFields {
			projection[field] = 1
		}

		findOptions = append(findOptions, options.Find().SetProjection(projection))
	}

	if len(m.ExcludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.ExcludeFields {
			projection[field] = 0
		}

		findOptions = append(findOptions, options.Find().SetProjection(projection))
	}

	cursor, err := client.Database(m.Database).Collection(m.Collection).Find(ctx, m.Filter, findOptions...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) FindOne(ctx context.Context, result any) error {

	var findOneOptions []options.Lister[options.FindOneOptions]

	if len(m.SortFields) > 0 {

		sort := bson.D{}
		for _, field := range m.SortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOneOptions = append(findOneOptions, options.FindOne().SetSort(sort))
	}

	if m.Index != "" {
		findOneOptions = append(findOneOptions, options.FindOne().SetHint(m.Index))
	}

	if len(m.IncludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.IncludeFields {
			projection[field] = 1
		}

		findOneOptions = append(findOneOptions, options.FindOne().SetProjection(projection))
	}

	if len(m.ExcludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.ExcludeFields {
			projection[field] = 0
		}

		findOneOptions = append(findOneOptions, options.FindOne().SetProjection(projection))
	}

	return client.Database(m.Database).Collection(m.Collection).FindOne(ctx, m.Filter, findOneOptions...).Decode(result)
}

func (m *MongoDB) FindByPage(ctx context.Context, paging *Paging, result any) (err error) {

	collection := client.Database(m.Database).Collection(m.Collection)

	if m.Filter == nil {
		paging.Total, err = collection.EstimatedDocumentCount(ctx)
	} else {
		paging.Total, err = collection.CountDocuments(ctx, m.Filter)
	}

	if err != nil {
		return err
	}

	paging.GetPages()

	var findOptions []options.Lister[options.FindOptions]

	findOptions = append(findOptions, options.Find().SetSkip(paging.StartNums).SetLimit(paging.PageSize))
	findOptions = append(findOptions, options.Find().SetAllowDiskUse(true))

	if len(m.SortFields) > 0 {

		sort := bson.D{}
		for _, field := range m.SortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOptions = append(findOptions, options.Find().SetSort(sort))
	}

	if m.Index != "" {
		findOptions = append(findOptions, options.Find().SetHint(m.Index))
	}

	if len(m.IncludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.IncludeFields {
			projection[field] = 1
		}

		findOptions = append(findOptions, options.Find().SetProjection(projection))
	}

	if len(m.ExcludeFields) > 0 {

		projection := make(map[string]any)
		for _, field := range m.ExcludeFields {
			projection[field] = 0
		}

		findOptions = append(findOptions, options.Find().SetProjection(projection))
	}

	cursor, err := collection.Find(ctx, m.Filter, findOptions...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) InsertOne(ctx context.Context, document any) (any, error) {

	oneResult, err := client.Database(m.Database).Collection(m.Collection).InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}

	return oneResult.InsertedID, nil
}

func (m *MongoDB) InsertMany(ctx context.Context, documents []any) ([]any, error) {

	manyResult, err := client.Database(m.Database).Collection(m.Collection).InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}

	return manyResult.InsertedIDs, nil
}

func (m *MongoDB) DeleteOne(ctx context.Context) (int64, error) {

	deleteResult, err := client.Database(m.Database).Collection(m.Collection).DeleteOne(ctx, m.Filter)
	if err != nil {
		return 0, err
	}

	return deleteResult.DeletedCount, nil
}

func (m *MongoDB) DeleteMany(ctx context.Context) (int64, error) {

	deleteResult, err := client.Database(m.Database).Collection(m.Collection).DeleteMany(ctx, m.Filter)
	if err != nil {
		return 0, err
	}

	return deleteResult.DeletedCount, nil
}

func (m *MongoDB) FindOneAndDelete(ctx context.Context, result any) error {
	return client.Database(m.Database).Collection(m.Collection).FindOneAndDelete(ctx, m.Filter).Decode(result)
}

func (m *MongoDB) UpdateById(ctx context.Context, id, update any, opts ...options.Lister[options.UpdateOneOptions]) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateByID(ctx, id, update, opts...)
	return err
}

func (m *MongoDB) UpdateOne(ctx context.Context, update any, opts ...options.Lister[options.UpdateOneOptions]) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateOne(ctx, m.Filter, update, opts...)
	return err
}

func (m *MongoDB) UpdateMany(ctx context.Context, update any, opts ...options.Lister[options.UpdateManyOptions]) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateMany(ctx, m.Filter, update, opts...)
	return err
}

func (m *MongoDB) FindOneAndUpdate(ctx context.Context, update any, result any, opts ...options.Lister[options.FindOneAndUpdateOptions]) error {
	return client.Database(m.Database).Collection(m.Collection).FindOneAndUpdate(ctx, m.Filter, update, opts...).Decode(result)
}

func (m *MongoDB) CountDocuments(ctx context.Context) (int64, error) {
	return client.Database(m.Database).Collection(m.Collection).CountDocuments(ctx, m.Filter)
}

func (m *MongoDB) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	return client.Database(m.Database).Collection(m.Collection).EstimatedDocumentCount(ctx)
}

func (m *MongoDB) Aggregate(ctx context.Context, result any, opts ...options.Lister[options.AggregateOptions]) error {

	cursor, err := client.Database(m.Database).Collection(m.Collection).Aggregate(ctx, m.Pipeline, opts...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) AggregateByPage(ctx context.Context, countResult, result any, opts ...options.Lister[options.AggregateOptions]) error {

	countCursor, err := client.Database(m.Database).Collection(m.Collection).Aggregate(ctx, m.CountPipeline, opts...)
	if err != nil {
		return err
	}

	if err = countCursor.All(ctx, countResult); err != nil {
		return err
	}

	cursor, err := client.Database(m.Database).Collection(m.Collection).Aggregate(ctx, m.Pipeline, opts...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}
