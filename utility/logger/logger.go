package logger

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
)

func Debug(ctx context.Context, v ...interface{}) {
	_ = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) { g.Log().Debug(ctx, v...) }, nil)
}

func Info(ctx context.Context, v ...interface{}) {
	g.Log().Info(ctx, v...)
}

func Error(ctx context.Context, v ...interface{}) {
	g.Log().Error(ctx, v...)
}

func Debugf(ctx context.Context, format string, v ...interface{}) {
	_ = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) { g.Log().Debugf(ctx, format, v...) }, nil)
}

func Infof(ctx context.Context, format string, v ...interface{}) {
	g.Log().Infof(ctx, format, v...)
}

func Errorf(ctx context.Context, format string, v ...interface{}) {
	g.Log().Errorf(ctx, format, v...)
}
