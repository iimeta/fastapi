package cmd

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/iimeta/fastapi/internal/controller/chat"
	"github.com/iimeta/fastapi/internal/controller/token"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/middleware"
	"net/http"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {

			s := g.Server()

			s.BindHookHandler("/*", ghttp.HookBeforeServe, beforeServeHook)

			s.Group("/", func(g *ghttp.RouterGroup) {
				g.Middleware(MiddlewareAuth)
				g.Middleware(MiddlewareHandlerResponse)
				g.Bind()
			})

			s.Group("/v1", func(v1 *ghttp.RouterGroup) {

				v1.Middleware(MiddlewareAuth)
				v1.Middleware(MiddlewareHandlerResponse)

				v1.Group("/chat", func(g *ghttp.RouterGroup) {
					g.Bind(
						chat.NewV1(),
					)
				})

				v1.Group("/token", func(g *ghttp.RouterGroup) {
					g.Bind(
						token.NewV1(),
					)
				})
			})

			s.Run()
			return nil
		},
	}
)

func beforeServeHook(r *ghttp.Request) {
	logger.Debugf(r.GetCtx(), "beforeServeHook [isFile: %v] URI: %s", r.IsFileRequest(), r.RequestURI)
	r.Response.CORSDefault()
}

func MiddlewareAuth(r *ghttp.Request) {
	middleware.Auth(r)
}

// DefaultHandlerResponse is the default implementation of HandlerResponse.
type DefaultHandlerResponse struct {
	Code    int         `json:"code"    dc:"Error code"`
	Message string      `json:"message" dc:"Error message"`
	Data    interface{} `json:"data"    dc:"Result data for certain request according API definition"`
}

// MiddlewareHandlerResponse is the default middleware handling handler response object and its error.
func MiddlewareHandlerResponse(r *ghttp.Request) {

	r.Middleware.Next()

	// There's custom buffer content, it then exits current handler.
	if r.Response.BufferLength() > 0 {
		return
	}

	var (
		msg  string
		err  = r.GetError()
		res  = r.GetHandlerResponse()
		code = gerror.Code(err)
	)
	if err != nil {
		if code == gcode.CodeNil {
			code = gcode.CodeInternalError
		}
		msg = err.Error()
	} else {
		if r.Response.Status > 0 && r.Response.Status != http.StatusOK {
			msg = http.StatusText(r.Response.Status)
			switch r.Response.Status {
			case http.StatusNotFound:
				code = gcode.CodeNotFound
			case http.StatusForbidden:
				code = gcode.CodeNotAuthorized
			default:
				code = gcode.CodeUnknown
			}
			// It creates error as it can be retrieved by other middlewares.
			err = gerror.NewCode(code, msg)
			r.SetError(err)
		} else {
			code = gcode.New(200, "success", "success")
			msg = code.Message()
		}
	}

	data := DefaultHandlerResponse{
		Code:    code.Code(),
		Message: msg,
		Data:    res,
	}

	logger.Debugf(r.GetCtx(), "url: %s, response body: %s", r.GetUrl(), gjson.MustEncodeString(data))

	r.Response.WriteJson(data)
}
