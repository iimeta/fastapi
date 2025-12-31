package redis

import (
	"context"
	"fmt"
	"time"

	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/redis/go-redis/v9"
)

var (
	UniversalClient redis.UniversalClient
	Client          *redis.Client
	master          *gredis.Redis
	slave           *gredis.Redis
)

type pipeliner redis.Pipeliner

func init() {

	ctx := gctx.New()

	redisConfig := gcfg.Instance().MustGet(ctx, "redis").MapStrVar()

	if def := redisConfig[gredis.DefaultGroupName]; def != nil {
		if err := gredis.SetConfigByMap(def.Map()); err != nil {
			panic(err)
		}
	}

	if masterConfig := redisConfig["master"]; masterConfig != nil {
		if err := gredis.SetConfigByMap(masterConfig.Map(), "master"); err != nil {
			panic(err)
		}
		if _, ok := gredis.GetConfig(); !ok {
			if err := gredis.SetConfigByMap(masterConfig.Map(), gredis.DefaultGroupName); err != nil {
				panic(err)
			}
		}
	}

	if slaveConfig := redisConfig["slave"]; slaveConfig != nil {
		if err := gredis.SetConfigByMap(slaveConfig.Map(), "slave"); err != nil {
			panic(err)
		}
		if _, ok := gredis.GetConfig(); !ok {
			if err := gredis.SetConfigByMap(slaveConfig.Map(), gredis.DefaultGroupName); err != nil {
				panic(err)
			}
		}
	}

	config, _ := gredis.GetConfig()

	opts := &redis.UniversalOptions{
		Addrs:            gstr.SplitAndTrim(config.Address, ","),
		Username:         config.User,
		Password:         config.Pass,
		SentinelUsername: config.User,
		SentinelPassword: config.Pass,
		DB:               config.Db,
		MaxRetries:       -1,
		PoolSize:         config.MaxActive,
		MinIdleConns:     config.MinIdle,
		MaxIdleConns:     config.MaxIdle,
		ConnMaxLifetime:  config.MaxConnLifetime,
		ConnMaxIdleTime:  config.IdleTimeout,
		PoolTimeout:      config.WaitTimeout,
		DialTimeout:      config.DialTimeout,
		ReadTimeout:      config.ReadTimeout,
		WriteTimeout:     config.WriteTimeout,
		MasterName:       config.MasterName,
		TLSConfig:        config.TLSConfig,
	}

	if opts.MasterName != "" {
		redisSentinel := opts.Failover()
		redisSentinel.ReplicaOnly = config.SlaveOnly
		UniversalClient = redis.NewFailoverClient(redisSentinel)
		Client = redis.NewFailoverClient(redisSentinel)
	} else if len(opts.Addrs) > 1 {
		UniversalClient = redis.NewClusterClient(opts.Cluster())
		Client = redis.NewClient(opts.Simple())
	} else {
		UniversalClient = redis.NewClient(opts.Simple())
		Client = redis.NewClient(opts.Simple())
	}

	master = g.Redis()
	if slave = gredis.Instance("slave"); slave == nil {
		slave = master
	}

	if cmd := Client.Ping(ctx); cmd.Err() != nil {
		panic(fmt.Sprint("Redis Client ", cmd.Err()))
	}

	logger.Info(ctx, "Redis Successfully connected and pinged.")
}

func Incr(ctx context.Context, key string) (int64, error) {
	return master.Incr(ctx, key)
}

func Set(ctx context.Context, key string, value any, option ...gredis.SetOption) (*gvar.Var, error) {
	return master.Set(ctx, key, value, option...)
}

func Get(ctx context.Context, key string) (*gvar.Var, error) {
	return slave.Get(ctx, key)
}

func GetInt(ctx context.Context, key string) (int, error) {
	reply, err := slave.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return reply.Int(), nil
}

func GetStr(ctx context.Context, key string) (string, error) {
	reply, err := slave.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return reply.String(), nil
}

func Del(ctx context.Context, keys ...string) (int64, error) {
	return master.Del(ctx, keys...)
}

func HSet(ctx context.Context, key string, fields map[string]any) (int64, error) {
	return master.HSet(ctx, key, fields)
}

func HGet(ctx context.Context, key, field string) (*gvar.Var, error) {
	return slave.HGet(ctx, key, field)
}

func HMGet(ctx context.Context, key string, fields ...string) (gvar.Vars, error) {
	return slave.HMGet(ctx, key, fields...)
}

func HVals(ctx context.Context, key string) (gvar.Vars, error) {
	return slave.HVals(ctx, key)
}

func HSetStrStr(ctx context.Context, key string, field, value string) (int64, error) {
	return HSet(ctx, key, g.MapStrAny{field: value})
}

func HSetStrAny(ctx context.Context, key string, field string, value any) (int64, error) {
	return HSet(ctx, key, g.MapStrAny{field: value})
}

func HGetStr(ctx context.Context, key, field string) (string, error) {
	reply, err := HGet(ctx, key, field)
	if err != nil {
		return "", err
	}
	return reply.String(), nil
}

func HGetInt(ctx context.Context, key, field string) (int, error) {
	reply, err := HGet(ctx, key, field)
	if err != nil {
		return 0, err
	}
	return reply.Int(), nil
}

func HIncrBy(ctx context.Context, key, field string, increment int64) (int64, error) {
	return master.HIncrBy(ctx, key, field, increment)
}

func HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	return master.HDel(ctx, key, fields...)
}

func SetEX(ctx context.Context, key string, value any, ttlInSeconds int64) error {
	return master.SetEX(ctx, key, value, ttlInSeconds)
}

func SetNX(ctx context.Context, key string, value any) (bool, error) {
	return master.SetNX(ctx, key, value)
}

func Expire(ctx context.Context, key string, seconds int64, option ...gredis.ExpireOption) (int64, error) {
	return master.Expire(ctx, key, seconds, option...)
}

func ExpireAt(ctx context.Context, key string, time time.Time, option ...gredis.ExpireOption) (int64, error) {
	return master.ExpireAt(ctx, key, time, option...)
}

func Pipeline(ctx context.Context) redis.Pipeliner {
	return Client.Pipeline()
}

func Pipelined(ctx context.Context, pipe pipeliner) ([]redis.Cmder, error) {
	return pipe.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner = pipe
		return nil
	})
}

func Publish(ctx context.Context, channel string, message any) (int64, error) {
	return master.Publish(ctx, config.Cfg.Core.ChannelPrefix+channel, message)
}

func Subscribe(ctx context.Context, channel string, channels ...string) (gredis.Conn, []*gredis.Subscription, error) {

	if len(channels) > 0 && config.Cfg.Core.ChannelPrefix != "" {
		for i, channel := range channels {
			if !gstr.HasPrefix(channel, config.Cfg.Core.ChannelPrefix) {
				channels[i] = config.Cfg.Core.ChannelPrefix + channel
			}
		}
	}

	if config.Cfg.Core.ChannelPrefix != "" && !gstr.HasPrefix(channel, config.Cfg.Core.ChannelPrefix) {
		channel = config.Cfg.Core.ChannelPrefix + channel
	}

	return slave.Subscribe(ctx, channel, channels...)
}

func PSubscribe(ctx context.Context, pattern string, patterns ...string) (gredis.Conn, []*gredis.Subscription, error) {
	return slave.PSubscribe(ctx, pattern, patterns...)
}

func RPush(ctx context.Context, key string, values ...any) (int64, error) {
	return master.RPush(ctx, key, values...)
}

func LPush(ctx context.Context, key string, values ...any) (int64, error) {
	return master.LPush(ctx, key, values...)
}

func LTrim(ctx context.Context, key string, start, stop int64) error {
	return master.LTrim(ctx, key, start, stop)
}

func LLen(ctx context.Context, key string) (int64, error) {
	return slave.LLen(ctx, key)
}

func LRange(ctx context.Context, key string, start, stop int64) (gvar.Vars, error) {
	return slave.LRange(ctx, key, start, stop)
}

func TTL(ctx context.Context, key string) (int64, error) {
	return slave.TTL(ctx, key)
}
