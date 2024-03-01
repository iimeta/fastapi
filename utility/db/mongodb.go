package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Database      string
	Collection    string
	Filter        map[string]interface{}
	CountPipeline []bson.M // AggregateByPage
	Pipeline      []bson.M // Aggregate/AggregateByPage
}

func (m *MongoDB) FindByPage(ctx context.Context, paging *Paging, result interface{}, sortFields ...string) (err error) {

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

	findOptions := []*options.FindOptions{{
		Skip:  &paging.StartNums,
		Limit: &paging.PageSize,
	}}

	if len(sortFields) > 0 {

		sort := bson.D{}
		for _, field := range sortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOptions = append(findOptions, &options.FindOptions{Sort: sort})
	}

	cursor, err := collection.Find(ctx, m.Filter, findOptions...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) Find(ctx context.Context, result interface{}, sortFields ...string) error {

	var findOptions []*options.FindOptions
	if len(sortFields) > 0 {

		sort := bson.D{}
		for _, field := range sortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOptions = append(findOptions, &options.FindOptions{Sort: sort})
	}

	cursor, err := client.Database(m.Database).Collection(m.Collection).Find(ctx, m.Filter, findOptions...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) FindOne(ctx context.Context, result interface{}, sortFields ...string) error {

	var findOneOptions []*options.FindOneOptions
	if len(sortFields) > 0 {

		sort := bson.D{}
		for _, field := range sortFields {
			if field[:1] == "-" {
				sort = append(sort, bson.E{Key: field[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: field, Value: 1})
			}
		}

		findOneOptions = append(findOneOptions, &options.FindOneOptions{Sort: sort})
	}

	return client.Database(m.Database).Collection(m.Collection).FindOne(ctx, m.Filter, findOneOptions...).Decode(result)
}

func (m *MongoDB) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {

	oneResult, err := client.Database(m.Database).Collection(m.Collection).InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}

	return oneResult.InsertedID, nil
}

func (m *MongoDB) InsertMany(ctx context.Context, documents []interface{}) ([]interface{}, error) {

	manyResult, err := client.Database(m.Database).Collection(m.Collection).InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}

	return manyResult.InsertedIDs, nil
}

func (m *MongoDB) DeleteById(ctx context.Context, id interface{}) error {
	return client.Database(m.Database).Collection(m.Collection).FindOneAndDelete(ctx, bson.M{"_id": id}).Err()
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

func (m *MongoDB) UpdateById(ctx context.Context, id, update interface{}, opts ...*options.UpdateOptions) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateByID(ctx, id, update, opts...)
	return err
}

func (m *MongoDB) UpdateOne(ctx context.Context, update interface{}, opts ...*options.UpdateOptions) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateOne(ctx, m.Filter, update, opts...)
	return err
}

func (m *MongoDB) UpdateMany(ctx context.Context, update interface{}, opts ...*options.UpdateOptions) error {
	_, err := client.Database(m.Database).Collection(m.Collection).UpdateMany(ctx, m.Filter, update, opts...)
	return err
}

func (m *MongoDB) CountDocuments(ctx context.Context) (int64, error) {
	return client.Database(m.Database).Collection(m.Collection).CountDocuments(ctx, m.Filter)
}

func (m *MongoDB) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	return client.Database(m.Database).Collection(m.Collection).EstimatedDocumentCount(ctx)
}

func (m *MongoDB) Aggregate(ctx context.Context, result interface{}, opts ...*options.AggregateOptions) error {

	cursor, err := client.Database(m.Database).Collection(m.Collection).Aggregate(ctx, m.Pipeline, opts...)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (m *MongoDB) AggregateByPage(ctx context.Context, countResult, result interface{}, opts ...*options.AggregateOptions) error {

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
