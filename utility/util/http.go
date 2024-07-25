package util

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/utility/logger"
	"net/http"
	"time"
)

func HttpGet(ctx context.Context, url string, header map[string]string, data g.Map, result interface{}, proxyURL ...string) error {

	logger.Infof(ctx, "HttpGet url: %s, header: %+v, data: %+v, proxyURL: %v", url, header, data, proxyURL)

	client := g.Client().Timeout(config.Cfg.Http.Timeout * time.Second)

	if header != nil {
		client.SetHeaderMap(header)
	}

	if len(proxyURL) > 0 {
		client.SetProxy(proxyURL[0])
	} else if config.Cfg.Http.ProxyUrl != "" {
		client.SetProxy(config.Cfg.Http.ProxyUrl)
	}

	response, err := client.Get(ctx, url, data)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	defer func() {
		if err = response.Close(); err != nil {
			logger.Error(ctx, err)
		}
	}()

	bytes := response.ReadAll()
	logger.Infof(ctx, "HttpGet url: %s, header: %+v, data: %+v, response: %s", url, header, data, string(bytes))

	if bytes != nil && len(bytes) > 0 {
		if err = gjson.Unmarshal(bytes, result); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

func HttpPost(ctx context.Context, url string, header map[string]string, data, result interface{}, proxyURL ...string) error {

	logger.Infof(ctx, "HttpPost url: %s, header: %+v, data: %+v, proxyURL: %v", url, header, data, proxyURL)

	client := g.Client().Timeout(config.Cfg.Http.Timeout * time.Second)

	if header != nil {
		client.SetHeaderMap(header)
	}

	if len(proxyURL) > 0 {
		client.SetProxy(proxyURL[0])
	} else if config.Cfg.Http.ProxyUrl != "" {
		client.SetProxy(config.Cfg.Http.ProxyUrl)
	}

	response, err := client.Post(ctx, url, data)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	defer func() {
		if err = response.Close(); err != nil {
			logger.Error(ctx, err)
		}
	}()

	bytes := response.ReadAll()
	logger.Infof(ctx, "HttpPost url: %s, header: %+v, data: %+v, response: %s", url, header, data, string(bytes))

	if bytes != nil && len(bytes) > 0 {
		if err = gjson.Unmarshal(bytes, result); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

func HttpPostJson(ctx context.Context, url string, header map[string]string, data, result interface{}, proxyURL ...string) error {

	logger.Infof(ctx, "HttpPostJson url: %s, header: %+v, data: %+v, proxyURL: %v", url, header, data, proxyURL)

	client := g.Client().Timeout(config.Cfg.Http.Timeout * time.Second)

	if header != nil {
		client.SetHeaderMap(header)
	}

	if len(proxyURL) > 0 {
		client.SetProxy(proxyURL[0])
	} else if config.Cfg.Http.ProxyUrl != "" {
		client.SetProxy(config.Cfg.Http.ProxyUrl)
	}

	response, err := client.ContentJson().Post(ctx, url, data)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	defer func() {
		if err = response.Close(); err != nil {
			logger.Error(ctx, err)
		}
	}()

	bytes := response.ReadAll()
	logger.Infof(ctx, "HttpPostJson url: %s, header: %+v, data: %+v, response: %s", url, header, data, string(bytes))

	if bytes != nil && len(bytes) > 0 {
		if err = gjson.Unmarshal(bytes, result); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

func HttpDownloadFile(ctx context.Context, fileURL string, proxyURL ...string) []byte {

	logger.Infof(ctx, "HttpDownloadFile fileURL: %s", fileURL)

	client := g.Client().Timeout(config.Cfg.Http.Timeout * time.Second)

	if len(proxyURL) > 0 {
		client.SetProxy(proxyURL[0])
	} else if config.Cfg.Http.ProxyUrl != "" {
		client.SetProxy(config.Cfg.Http.ProxyUrl)
	}

	return client.GetBytes(ctx, fileURL)
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
		logger.Error(ctx, err)
		return err
	}

	flusher.Flush()

	return nil
}
