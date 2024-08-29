package util

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/utility/logger"
	"net/http"
	"time"
)

func HttpPost(ctx context.Context, url string, header map[string]string, data, result interface{}, proxyURL string) error {

	logger.Infof(ctx, "HttpPost url: %s, header: %+v, data: %s, proxyURL: %s", url, header, gjson.MustEncodeString(data), proxyURL)

	client := g.Client().Timeout(config.Cfg.Http.Timeout * time.Second)

	if header != nil {
		client.SetHeaderMap(header)
	}

	if proxyURL != "" {
		client.SetProxy(proxyURL)
	}

	response, err := client.Post(ctx, url, data)
	if response != nil {
		defer func() {
			if err := response.Close(); err != nil {
				logger.Error(ctx, err)
			}
		}()
	}

	if err != nil {
		logger.Errorf(ctx, "HttpPost url: %s, header: %+v, data: %s, proxyURL: %s, err: %v", url, header, gjson.MustEncodeString(data), proxyURL, err)
		return err
	}

	bytes := response.ReadAll()
	logger.Infof(ctx, "HttpPost url: %s, statusCode: %d, header: %+v, data: %s, proxyURL: %s, response: %s", url, response.StatusCode, header, gjson.MustEncodeString(data), proxyURL, string(bytes))

	if bytes != nil && len(bytes) > 0 {
		if err = gjson.Unmarshal(bytes, result); err != nil {
			logger.Errorf(ctx, "HttpPost url: %s, statusCode: %d, header: %+v, data: %s, proxyURL: %s, response: %s, err: %v", url, response.StatusCode, header, gjson.MustEncodeString(data), proxyURL, string(bytes), err)
			return errors.Newf("response: %s, err: %v", bytes, err)
		}
	}

	return nil
}

func SSEServer(ctx context.Context, data string) error {

	r := g.RequestFromCtx(ctx)
	rw := r.Response.RawWriter()
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported", http.StatusInternalServerError)
		return gerror.New("Streaming unsupported")
	}

	r.Response.Header().Set("Trace-Id", gctx.CtxId(ctx))
	r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("Connection", "keep-alive")

	if _, err := fmt.Fprintf(rw, "data: %s\n\n", data); err != nil {
		logger.Errorf(ctx, "SSEServer data: %s, err: %v", data, err)
		return err
	}

	flusher.Flush()

	return nil
}
