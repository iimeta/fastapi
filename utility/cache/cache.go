package cache

import (
	"context"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/os/gcache"
	"time"
)

type Cache struct {
	cache *gcache.Cache
}

func New(lruCap ...int) *Cache {
	return &Cache{
		cache: gcache.New(lruCap...),
	}
}

func (c *Cache) Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error {
	return c.cache.Set(ctx, key, value, duration)
}

func (c *Cache) SetMap(ctx context.Context, data map[interface{}]interface{}, duration time.Duration) error {
	return c.cache.SetMap(ctx, data, duration)
}

func (c *Cache) SetIfNotExist(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (bool, error) {
	return c.cache.SetIfNotExist(ctx, key, value, duration)
}

func (c *Cache) SetIfNotExistFunc(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (bool, error) {
	return c.cache.SetIfNotExistFunc(ctx, key, f, duration)
}

func (c *Cache) SetIfNotExistFuncLock(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (bool, error) {
	return c.cache.SetIfNotExistFuncLock(ctx, key, f, duration)
}

func (c *Cache) Get(ctx context.Context, key interface{}) (*gvar.Var, error) {
	return c.cache.Get(ctx, key)
}

func (c *Cache) GetVal(ctx context.Context, key interface{}) interface{} {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil
	}
	return reply.Val()
}

func (c *Cache) GetInt(ctx context.Context, key interface{}) (int, error) {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return reply.Int(), nil
}

func (c *Cache) GetStr(ctx context.Context, key interface{}) (string, error) {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return reply.String(), nil
}

func (c *Cache) GetMap(ctx context.Context, key interface{}) (map[string]interface{}, error) {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.Map(), nil
}

func (c *Cache) GetMapStrStr(ctx context.Context, key interface{}) (map[string]string, error) {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.MapStrStr(), nil
}

func (c *Cache) GetMapStrVar(ctx context.Context, key interface{}) (map[string]*gvar.Var, error) {
	reply, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return reply.MapStrVar(), nil
}

func (c *Cache) GetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (*gvar.Var, error) {
	return c.cache.GetOrSet(ctx, key, value, duration)
}

func (c *Cache) GetOrSetFunc(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (*gvar.Var, error) {
	return c.cache.GetOrSetFunc(ctx, key, f, duration)
}

func (c *Cache) GetOrSetFuncLock(ctx context.Context, key interface{}, f gcache.Func, duration time.Duration) (*gvar.Var, error) {
	return c.cache.GetOrSetFuncLock(ctx, key, f, duration)
}

func (c *Cache) Contains(ctx context.Context, key interface{}) (bool, error) {
	return c.cache.Contains(ctx, key)
}

func (c *Cache) ContainsKey(ctx context.Context, key interface{}) bool {
	ok, err := c.cache.Contains(ctx, key)
	if err != nil {
		return false
	}
	return ok
}

func (c *Cache) Size(ctx context.Context) (int, error) {
	return c.cache.Size(ctx)
}

func (c *Cache) Data(ctx context.Context) (map[interface{}]interface{}, error) {
	return c.cache.Data(ctx)
}

func (c *Cache) Keys(ctx context.Context) ([]interface{}, error) {
	return c.cache.Keys(ctx)
}

func (c *Cache) Values(ctx context.Context) ([]interface{}, error) {
	return c.cache.Values(ctx)
}

func (c *Cache) Update(ctx context.Context, key interface{}, value interface{}) (oldValue *gvar.Var, exist bool, err error) {
	return c.cache.Update(ctx, key, value)
}

func (c *Cache) UpdateExpire(ctx context.Context, key interface{}, duration time.Duration) (oldDuration time.Duration, err error) {
	return c.cache.UpdateExpire(ctx, key, duration)
}

func (c *Cache) GetExpire(ctx context.Context, key interface{}) (time.Duration, error) {
	return c.cache.GetExpire(ctx, key)
}

func (c *Cache) Remove(ctx context.Context, keys ...interface{}) (lastValue *gvar.Var, err error) {
	return c.cache.Remove(ctx, keys...)
}

func (c *Cache) Clear(ctx context.Context) error {
	return c.cache.Clear(ctx)
}

func (c *Cache) Close(ctx context.Context) error {
	return c.cache.Close(ctx)
}
