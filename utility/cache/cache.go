package cache

import (
	"context"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/os/gcache"
	"time"
)

var (
	Cache *gcache.Cache
)

func init() {
	Cache = gcache.New()
}

func Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error {
	return Cache.Set(ctx, key, value, duration)
}

func SetMap(ctx context.Context, data map[interface{}]interface{}, duration time.Duration) error {
	return Cache.SetMap(ctx, data, duration)
}

func SetIfNotExist(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (bool, error) {
	return Cache.SetIfNotExist(ctx, key, value, duration)
}

func SetIfNotExistFunc(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (bool, error) {
	return Cache.SetIfNotExistFunc(ctx, key, f, duration)
}

func SetIfNotExistFuncLock(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (bool, error) {
	return Cache.SetIfNotExistFuncLock(ctx, key, f, duration)
}

func Get(ctx context.Context, key interface{}) (*gvar.Var, error) {
	return Cache.Get(ctx, key)
}

func GetInt(ctx context.Context, key interface{}) (int, error) {
	reply, err := Cache.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return reply.Int(), nil
}

func GetStr(ctx context.Context, key interface{}) (string, error) {
	reply, err := Cache.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return reply.String(), nil
}

func GetMap(ctx context.Context, key interface{}) (map[string]interface{}, error) {
	reply, err := Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.Map(), nil
}

func GetMapStrStr(ctx context.Context, key interface{}) (map[string]string, error) {
	reply, err := Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.MapStrStr(), nil
}

func GetMapStrVar(ctx context.Context, key interface{}) (map[string]*gvar.Var, error) {
	reply, err := Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.MapStrVar(), nil
}

func GetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (*gvar.Var, error) {
	return Cache.GetOrSet(ctx, key, value, duration)
}

func GetOrSetFunc(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (*gvar.Var, error) {
	return Cache.GetOrSetFunc(ctx, key, f, duration)
}

func GetOrSetFuncLock(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (*gvar.Var, error) {
	return Cache.GetOrSetFuncLock(ctx, key, f, duration)
}

func Contains(ctx context.Context, key interface{}) (bool, error) {
	return Cache.Contains(ctx, key)
}

func Size(ctx context.Context) (int, error) {
	return Cache.Size(ctx)
}

func Data(ctx context.Context) (map[interface{}]interface{}, error) {
	return Cache.Data(ctx)
}

func Keys(ctx context.Context) ([]interface{}, error) {
	return Cache.Keys(ctx)
}

func Values(ctx context.Context) ([]interface{}, error) {
	return Cache.Values(ctx)
}

func Update(ctx context.Context, key interface{}, value interface{}) (oldValue *gvar.Var, exist bool, err error) {
	return Cache.Update(ctx, key, value)
}

func UpdateExpire(ctx context.Context, key interface{}, duration time.Duration) (oldDuration time.Duration, err error) {
	return Cache.UpdateExpire(ctx, key, duration)
}

func GetExpire(ctx context.Context, key interface{}) (time.Duration, error) {
	return Cache.GetExpire(ctx, key)
}

func Remove(ctx context.Context, keys ...interface{}) (lastValue *gvar.Var, err error) {
	return Cache.Remove(ctx, keys...)
}

func Clear(ctx context.Context) error {
	return Cache.Clear(ctx)
}

func Close(ctx context.Context) error {
	return Cache.Close(ctx)
}
