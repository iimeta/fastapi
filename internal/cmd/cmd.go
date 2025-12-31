package cmd

import (
	"context"
	"net/http"
	"runtime"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/controller/anthropic"
	"github.com/iimeta/fastapi/v2/internal/controller/audio"
	"github.com/iimeta/fastapi/v2/internal/controller/batch"
	"github.com/iimeta/fastapi/v2/internal/controller/chat"
	"github.com/iimeta/fastapi/v2/internal/controller/dashboard"
	"github.com/iimeta/fastapi/v2/internal/controller/embedding"
	"github.com/iimeta/fastapi/v2/internal/controller/file"
	"github.com/iimeta/fastapi/v2/internal/controller/general"
	"github.com/iimeta/fastapi/v2/internal/controller/google"
	"github.com/iimeta/fastapi/v2/internal/controller/health"
	"github.com/iimeta/fastapi/v2/internal/controller/image"
	"github.com/iimeta/fastapi/v2/internal/controller/midjourney"
	"github.com/iimeta/fastapi/v2/internal/controller/moderation"
	"github.com/iimeta/fastapi/v2/internal/controller/openai"
	"github.com/iimeta/fastapi/v2/internal/controller/video"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {

			s := g.Server()

			if config.Cfg.Debug.Open {
				runtime.SetMutexProfileFraction(1) // (非必需)开启对锁调用的跟踪
				runtime.SetBlockProfileRate(1)     // (非必需)开启对阻塞操作的跟踪
				s.EnablePProf()
			}

			s.BindHookHandler("/*", ghttp.HookBeforeServe, beforeServeHook)

			s.SetServerRoot("./resource/index/")

			s.Group("/", func(g *ghttp.RouterGroup) {
				g.Middleware(middlewareHandlerResponse)
				g.Bind(
					health.NewV1(),
				)
			})

			s.BindHandler("/v1/realtime", func(r *ghttp.Request) {
				middleware(r)
				if err := service.Realtime().Realtime(r.GetCtx(), r, model.RealtimeRequest{
					Model: r.FormValue("model"),
				}, nil, nil); err != nil {
					err := errors.Error(r.GetCtx(), err)
					r.Response.Header().Set("Content-Type", "application/json")
					r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
					r.Exit()
				}
			})

			s.Group("/v1", func(v1 *ghttp.RouterGroup) {

				v1.Middleware(middlewareHandlerResponse)
				v1.Middleware(middleware)

				v1.Group("/", func(g *ghttp.RouterGroup) {
					g.Bind(
						dashboard.NewV1(),
						openai.NewV1(),
						anthropic.NewV1(),
						google.NewV1(),
						embedding.NewV1(),
						moderation.NewV1(),
						general.NewV1(),
					)
				})

				v1.Group("/dashboard", func(g *ghttp.RouterGroup) {
					g.Bind(
						dashboard.NewV1(),
					)
				})

				v1.Group("/chat", func(g *ghttp.RouterGroup) {
					g.Bind(
						chat.NewV1(),
					)
				})

				v1.Group("/images", func(g *ghttp.RouterGroup) {
					g.Bind(
						image.NewV1(),
					)
				})

				v1.Group("/audio", func(g *ghttp.RouterGroup) {
					g.Bind(
						audio.NewV1(),
					)
				})

				v1.Group("/videos", func(g *ghttp.RouterGroup) {
					g.Bind(
						video.NewV1(),
					)
				})

				v1.Group("/files", func(g *ghttp.RouterGroup) {
					g.Bind(
						file.NewV1(),
					)
				})

				v1.Group("/batches", func(g *ghttp.RouterGroup) {
					g.Bind(
						batch.NewV1(),
					)
				})
			})

			s.Group("/v1beta", func(v1 *ghttp.RouterGroup) {
				v1.Middleware(middlewareHandlerResponse)
				v1.Middleware(middleware)
				v1.Bind(
					google.NewV1(),
				)
			})

			s.Group("/mj**", func(v1 *ghttp.RouterGroup) {
				v1.Middleware(middlewareHandlerResponse)
				v1.Middleware(middleware)
				v1.Bind(
					midjourney.NewV1(),
				)
			})

			s.Group("/{provider}/v1", func(v1 *ghttp.RouterGroup) {

				v1.Middleware(middlewareHandlerResponse)
				v1.Middleware(middleware)

				v1.Group("/files", func(g *ghttp.RouterGroup) {
					g.Bind(
						file.NewV1(),
					)
				})

				v1.Group("/batches", func(g *ghttp.RouterGroup) {
					g.Bind(
						batch.NewV1(),
					)
				})
			})

			if config.Cfg.ApiServerAddress != "" {
				s.SetAddr(config.Cfg.ApiServerAddress)
			}

			s.Run()
			return nil
		},
	}
)

func beforeServeHook(r *ghttp.Request) {

	ctx := gctx.WithSpan(r.GetCtx(), "gctx.WithSpan")

	if traceId := r.Header.Get(consts.TRACE_ID); traceId != "" {
		ctx, _ = gtrace.WithTraceID(ctx, traceId)
	}

	r.SetCtx(ctx)
	r.SetCtxVar(consts.HOST_KEY, r.GetHost())

	logger.Infof(r.GetCtx(), "beforeServeHook ClientIp: %s, RemoteIp: %s, IsFile: %t, URI: %s", r.GetClientIp(), r.GetRemoteIp(), r.IsFileRequest(), r.RequestURI)

	r.Response.CORSDefault()
}

func middleware(r *ghttp.Request) {

	if config.Cfg.ServiceUnavailable.Open && !slices.Contains(config.Cfg.ServiceUnavailable.IpWhitelist, r.GetClientIp()) {
		err := errors.Error(r.GetCtx(), errors.ERR_SERVICE_UNAVAILABLE)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
		r.Exit()
		return
	}

	logger.Debugf(r.GetCtx(), "r.Header: %v", r.Header)

	secretKey := strings.TrimPrefix(r.GetHeader("Authorization"), "Bearer ")

	if secretKey == "" {
		if key := r.Get("key"); key != nil {
			secretKey = key.String()
		}
	}

	if secretKey == "" {
		if token := r.Get("token"); token != nil {
			secretKey = token.String()
		}
	}

	if secretKey == "" {
		secretKey = r.Header.Get("x-api-key")
	}

	if secretKey == "" {
		secretKey = r.Header.Get("api-key")
	}

	if secretKey == "" {
		secretKey = r.Header.Get("x-goog-api-key")
	}

	if secretKey == "" {
		secretKey = r.GetHeader(config.Cfg.Midjourney.ApiSecretHeader)
	}

	if secretKey == "" {
		swp := r.Header.Values("Sec-Websocket-Protocol")
		if len(swp) > 0 {
			values := gstr.Split(swp[0], ", ")
			for _, value := range values {
				if gstr.HasPrefix(value, "openai-insecure-api-key") {
					split := gstr.Split(value, ".")
					if len(split) == 2 {
						secretKey = split[1]
					} else {
						secretKey = value
					}
				}
			}
		}
	}

	if secretKey == "" {
		err := errors.Error(r.GetCtx(), errors.ERR_NOT_API_KEY)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
		r.Exit()
		return
	}

	logger.Infof(r.GetCtx(), "middleware secretKey: %s", secretKey)

	if err := service.Auth().Authenticator(r.GetCtx(), secretKey); err != nil {
		err := errors.Error(r.GetCtx(), err)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
		r.Exit()
		return
	}

	if config.Cfg.Debug.Open {
		if gstr.HasPrefix(r.GetHeader("Content-Type"), "application/json") {
			logger.Debugf(r.GetCtx(), "middleware url: %s, request body: %s", r.GetUrl(), r.GetBodyString())
		} else {
			logger.Debugf(r.GetCtx(), "middleware url: %s, Content-Type: %s", r.GetUrl(), r.GetHeader("Content-Type"))
		}
	}

	r.Middleware.Next()
}

type defaultHandlerResponse struct {
	Code    any    `json:"code"    dc:"Error code"`
	Message string `json:"message" dc:"Error message"`
	Data    any    `json:"data"    dc:"Result data for certain request according API definition"`
}

func middlewareHandlerResponse(r *ghttp.Request) {

	r.Middleware.Next()

	// There's custom buffer content, it then exits current handler.
	if r.Response.BufferLength() > 0 {
		if config.Cfg.Debug.Open {
			if gstr.HasPrefix(r.Response.Header().Get("Content-Type"), "application/json") {
				logger.Debugf(r.GetCtx(), "middlewareHandlerResponse url: %s, response: %s", r.GetUrl(), r.Response.BufferString())
			}
		}
		return
	}

	var (
		msg  string
		err  = r.GetError()
		res  = r.GetHandlerResponse()
		code = errors.Error(r.GetCtx(), err)
	)

	if err != nil {

		if code == errors.Error(r.GetCtx(), errors.ERR_NIL) {
			code = errors.Error(r.GetCtx(), errors.ERR_INTERNAL_ERROR)
		}

		msg = err.Error()

		if gstr.Contains(msg, "timeout") || gstr.Contains(msg, "tcp") || gstr.Contains(msg, "http") ||
			gstr.Contains(msg, "connection") || gstr.Contains(msg, "failed") {
			msg = "The server is busy, please try again later"
		}

	} else {

		if r.Response.Status > 0 && r.Response.Status != http.StatusOK {

			msg = http.StatusText(r.Response.Status)

			switch r.Response.Status {
			case http.StatusNotFound:
				code = errors.Error(r.GetCtx(), errors.ERR_NOT_FOUND)
			case http.StatusForbidden:
				code = errors.Error(r.GetCtx(), errors.ERR_NOT_AUTHORIZED)
			default:
				code = errors.Error(r.GetCtx(), errors.ERR_UNKNOWN)
			}

			err = code.Unwrap()
			r.SetError(err)

		} else {
			code = errors.Error(r.GetCtx(), errors.NewError(200, 0, "success", "success", nil))
			msg = code.ErrMessage()
		}
	}

	if err != nil {

		err := errors.Error(r.GetCtx(), err)

		if config.Cfg.Debug.Open {
			logger.Debugf(r.GetCtx(), "middlewareHandlerResponse url: %s, response: %s", r.GetUrl(), gjson.MustEncodeString(err))
		}

		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))

	} else {

		stream := r.GetCtxVar("stream")
		if stream == nil || !stream.Bool() {

			content := defaultHandlerResponse{
				Code:    code.ErrCode(),
				Message: msg,
				Data:    res,
			}

			if config.Cfg.Debug.Open {
				logger.Debugf(r.GetCtx(), "middlewareHandlerResponse url: %s, response: %s", r.GetUrl(), gjson.MustEncodeString(content))
			}

			r.Response.WriteJson(content)
		}
	}
}
