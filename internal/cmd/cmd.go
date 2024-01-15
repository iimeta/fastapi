package cmd

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/controller/chat"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"net/http"
	"strings"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {

			s := g.Server()

			s.BindHookHandler("/*", ghttp.HookBeforeServe, beforeServeHook)

			s.Group("/", func(r *ghttp.RouterGroup) {
				r.Bind(
					func(r *ghttp.Request) {
						r.Response.WriteStatus(http.StatusOK, "Hello FastAPI")
						r.Exit()
						return
					},
				)
			})

			s.Group("/v1", func(v1 *ghttp.RouterGroup) {

				v1.Middleware(middleware)
				v1.Middleware(middlewareHandlerResponse)

				v1.Group("/chat", func(g *ghttp.RouterGroup) {
					g.Bind(
						chat.NewV1(),
					)
				})
			})

			s.Run()
			return nil
		},
	}
)

func beforeServeHook(r *ghttp.Request) {
	logger.Debugf(r.GetCtx(), "beforeServeHook [isFile: %t] URI: %s", r.IsFileRequest(), r.RequestURI)
	r.Response.CORSDefault()
}

func middleware(r *ghttp.Request) {

	secretKey := strings.TrimPrefix(r.GetHeader("Authorization"), "Bearer ")
	if secretKey == "" {
		err := errors.Error(errors.ERR_NOT_API_KEY)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
		r.Exit()
		return
	}

	if pass, err := service.Auth().VerifySecretKey(r.GetCtx(), secretKey); err != nil || !pass {
		err := errors.Error(err)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))
		r.Exit()
		return
	}

	if gstr.HasPrefix(r.GetHeader("Content-Type"), "application/json") {
		logger.Debugf(r.GetCtx(), "url: %s, request body: %s", r.GetUrl(), r.GetBodyString())
	} else {
		logger.Debugf(r.GetCtx(), "url: %s, Content-Type: %s", r.GetUrl(), r.GetHeader("Content-Type"))
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
		code = errors.Error(err)
	)
	if err != nil {
		if code == errors.Error(errors.ERR_NIL) {
			code = errors.Error(errors.ERR_INTERNAL_ERROR)
		}
		msg = err.Error()
	} else {
		if r.Response.Status > 0 && r.Response.Status != http.StatusOK {

			msg = http.StatusText(r.Response.Status)

			switch r.Response.Status {
			case http.StatusNotFound:
				code = errors.Error(errors.ERR_NOT_FOUND)
			case http.StatusForbidden:
				code = errors.Error(errors.ERR_NOT_AUTHORIZED)
			default:
				code = errors.Error(errors.ERR_UNKNOWN)
			}

			err = code.Unwrap()
			r.SetError(err)

		} else {
			code = errors.Error(errors.NewError(200, 0, "success", "success"))
			msg = code.ErrMessage()
		}
	}

	if err != nil {

		logger.Debugf(r.GetCtx(), "url: %s, response body: %s", r.GetUrl(), gjson.MustEncodeString(err))

		err := errors.Error(err)
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteStatus(err.Status(), gjson.MustEncodeString(err))

	} else {

		data := defaultHandlerResponse{
			Code:    code.ErrCode(),
			Message: msg,
			Data:    res,
		}

		logger.Debugf(r.GetCtx(), "url: %s, response body: %s", r.GetUrl(), gjson.MustEncodeString(data))

		r.Response.WriteJson(data)
	}
}
