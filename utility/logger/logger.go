package logger

import (
	"context"
	"encoding/json"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
)

func Debug(ctx context.Context, v ...any) {
	_ = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		var compactV []any
		for _, item := range v {
			switch val := item.(type) {
			case string:
				var jsonObj any
				if err := json.Unmarshal([]byte(val), &jsonObj); err == nil {
					if compactJSON, err := json.Marshal(jsonObj); err == nil {
						compactV = append(compactV, string(compactJSON))
					} else {
						compactV = append(compactV, val)
					}
				} else {
					compactV = append(compactV, val)
				}
			default:
				compactV = append(compactV, item)
			}
		}
		g.Log().Debug(ctx, compactV...)
	}, nil)
}

func Info(ctx context.Context, v ...any) {
	g.Log().Info(ctx, v...)
}

func Error(ctx context.Context, v ...any) {
	g.Log().Error(ctx, v...)
}

func Debugf(ctx context.Context, format string, v ...any) {
	_ = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		var compactV []any
		for _, item := range v {
			switch val := item.(type) {
			case string:
				var jsonObj any
				if err := json.Unmarshal([]byte(val), &jsonObj); err == nil {
					if compactJSON, err := json.Marshal(jsonObj); err == nil {
						compactV = append(compactV, string(compactJSON))
					} else {
						compactV = append(compactV, val)
					}
				} else {
					compactV = append(compactV, val)
				}
			default:
				compactV = append(compactV, item)
			}
		}
		g.Log().Debugf(ctx, format, compactV...)
	}, nil)
}

func Infof(ctx context.Context, format string, v ...any) {
	g.Log().Infof(ctx, format, v...)
}

func Errorf(ctx context.Context, format string, v ...any) {
	g.Log().Errorf(ctx, format, v...)
}
