package cmd

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/controller/audio"
	"github.com/iimeta/fastapi/internal/controller/chat"
	"github.com/iimeta/fastapi/internal/controller/dashboard"
	"github.com/iimeta/fastapi/internal/controller/embedding"
	"github.com/iimeta/fastapi/internal/controller/health"
	"github.com/iimeta/fastapi/internal/controller/image"
	"github.com/iimeta/fastapi/internal/controller/midjourney"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"net/http"
	"runtime"
	"strings"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {

			runtime.SetMutexProfileFraction(1) // (非必需)开启对锁调用的跟踪
			runtime.SetBlockProfileRate(1)     // (非必需)开启对阻塞操作的跟踪

			s := g.Server()
			s.EnablePProf()

			s.BindHookHandler("/*", ghttp.HookBeforeServe, beforeServeHook)

			s.Group("/", func(g *ghttp.RouterGroup) {

				g.Middleware(middlewareHandlerResponse)

				g.Bind(
					func(r *ghttp.Request) {
						r.Response.WriteStatus(http.StatusOK, "Hello Fast API")
						r.Exit()
						return
					},
					health.NewV1(),
				)
			})

			s.BindHandler("/v1/realtime", func(r *ghttp.Request) {
				middleware(r)
				if err := service.Realtime().Realtime(r); err != nil {
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
						embedding.NewV1(),
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
			})

			s.Group("/mj**", func(v1 *ghttp.RouterGroup) {
				v1.Middleware(middlewareHandlerResponse)
				v1.Middleware(middleware)
				v1.Bind(
					midjourney.NewV1(),
				)
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
	r.SetCtxVar(consts.HOST_KEY, r.GetHost())
	logger.Infof(r.GetCtx(), "beforeServeHook ClientIp: %s, RemoteIp: %s, IsFile: %t, URI: %s", r.GetClientIp(), r.GetRemoteIp(), r.IsFileRequest(), r.RequestURI)
	r.Response.CORSDefault()
}

func middleware(r *ghttp.Request) {

	secretKey := strings.TrimPrefix(r.GetHeader("Authorization"), "Bearer ")
	if secretKey == "" {
		secretKey = r.GetHeader(config.Cfg.Midjourney.MidjourneyProxy.ApiSecretHeader)
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

	if config.Cfg.Debug {
		if gstr.HasPrefix(r.GetHeader("Content-Type"), "application/json") {
			logger.Debugf(r.GetCtx(), "url: %s, request body: %s", r.GetUrl(), r.GetBodyString())
		} else {
			logger.Debugf(r.GetCtx(), "url: %s, Content-Type: %s", r.GetUrl(), r.GetHeader("Content-Type"))
		}
	}

	r.Middleware.Next()
}

type defaultHandlerResponse struct {
	Code    any         `json:"code"    dc:"Error code"`
	Message string      `json:"message" dc:"Error message"`
	Data    interface{} `json:"data"    dc:"Result data for certain request according API definition"`
}

func middlewareHandlerResponse(r *ghttp.Request) {

	r.Middleware.Next()

	// There's custom buffer content, it then exits current handler.
	if r.Response.BufferLength() > 0 {
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
			code = errors.Error(r.GetCtx(), errors.NewError(200, 0, "success", "success"))
			msg = code.ErrMessage()
		}
	}

	if err != nil {
		err := errors.Error(r.GetCtx(), err)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
	} else {
		stream := r.GetCtxVar("stream")
		if stream == nil || !stream.Bool() {
			r.Response.WriteJson(defaultHandlerResponse{
				Code:    code.ErrCode(),
				Message: msg,
				Data:    res,
			})
		}
	}
}
