package util

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func SSEServer(ctx context.Context, data string, event ...string) error {

	r := g.RequestFromCtx(ctx)
	rw := r.Response.RawWriter()
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported", http.StatusInternalServerError)
		return errors.New("Streaming unsupported")
	}

	r.Response.Header().Set(consts.TRACE_ID, gtrace.GetTraceID(ctx))
	r.Response.Header().Set(consts.FASTAPI_REQUEST_ID, gtrace.GetTraceID(ctx))
	r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("Connection", "keep-alive")
	r.Response.Header().Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	flusher.Flush()

	var builder strings.Builder

	if len(event) > 0 && event[0] != "" {
		builder.WriteString("event: ")
		builder.WriteString(event[0])
		builder.WriteString("\n")
	}

	builder.WriteString("data: ")
	builder.WriteString(data)
	builder.WriteString("\n\n")

	if _, err := io.WriteString(rw, builder.String()); err != nil {
		if !config.Cfg.Base.AllowRequestAbort && isAborted(err) {
			return nil
		}
		logger.Errorf(ctx, "SSEServer data: %s, error: %v", builder.String(), err)
		return err
	}

	flusher.Flush()

	return nil
}

func isAborted(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) ||
		gstr.Contains(err.Error(), "context deadline exceeded") ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}
