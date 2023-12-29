package logger

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
)

func Debug(ctx context.Context, v ...interface{}) {
	g.Log().Debug(ctx, v...)
}

func Info(ctx context.Context, v ...interface{}) {
	g.Log().Info(ctx, v...)
}

func Error(ctx context.Context, v ...interface{}) {
	g.Log().Error(ctx, v...)
}

func Debugf(ctx context.Context, format string, v ...interface{}) {
	g.Log().Debugf(ctx, format, v...)
}

func Infof(ctx context.Context, format string, v ...interface{}) {
	g.Log().Infof(ctx, format, v...)
}

func Errorf(ctx context.Context, format string, v ...interface{}) {
	g.Log().Errorf(ctx, format, v...)
}
