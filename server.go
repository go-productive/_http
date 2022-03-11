package _http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type (
	Server struct {
		Engine  *gin.Engine
		options *_Options
	}
)

func New(opts ...Option) *Server {
	options := newOptions(opts...)
	s := &Server{
		Engine:  options.engine,
		options: options,
	}
	if s.Engine == nil {
		s.Engine = gin.New()
		s.Engine.Use(gin.Recovery())
	}
	return s
}

func (s *Server) GinHandlerFunc(reqFunc func() interface{}, handleFunc HandleFunc) gin.HandlerFunc {
	for i := len(s.options.handleFuncWraps) - 1; i >= 0; i-- {
		handleFunc = s.options.handleFuncWraps[i](handleFunc)
	}
	return func(ginCtx *gin.Context) {
		req := reqFunc()
		if err := s.options.bindReqFunc(req, ginCtx); err != nil {
			ginCtx.Header(s.options.errorHeader, err.Error())
			ginCtx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ctx := s.options.ctxFunc(ginCtx)
		rsp, err := handleFunc(req, ctx)
		if ctx, ok := ctx.(*DefaultContext); ok {
			ginCtx.Header("trace-id", ctx.TraceID)
		}
		if rsp, ok := rsp.(WithHeader); ok {
			for k, v := range rsp.Header() {
				ginCtx.Header(k, v[0])
			}
		}
		code := http.StatusOK
		if err != nil {
			switch err := err.(type) {
			case HTTPError:
				code = err.StatusCode
			default:
				code = http.StatusInternalServerError
			}
			ginCtx.Header(s.options.errorHeader, err.Error())
		}
		ginCtx.JSON(code, rsp)
	}
}
