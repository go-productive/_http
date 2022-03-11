package _http

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"sync/atomic"
	"time"
)

type (
	DefaultContext struct {
		TraceID string
		Ctx     context.Context
		GinCtx  *gin.Context
	}
	_Options struct {
		engine          *gin.Engine
		errorHeader     string
		handleFuncWraps []HandleFuncWrap
		bindReqFunc     func(req interface{}, ginCtx *gin.Context) error
		ctxFunc         func(ginCtx *gin.Context) interface{}
	}
	Option         func(*_Options)
	HandleFunc     func(req interface{}, ctx interface{}) (rsp interface{}, err error)
	HandleFuncWrap func(HandleFunc) HandleFunc
)

func newOptions(opts ...Option) *_Options {
	o := &_Options{
		errorHeader: "ERROR",
		bindReqFunc: func(req interface{}, ginCtx *gin.Context) error {
			return ginCtx.ShouldBind(req)
		},
		ctxFunc: func(ginCtx *gin.Context) interface{} {
			return &DefaultContext{
				TraceID: mongoObjectID(),
				Ctx:     context.TODO(),
				GinCtx:  ginCtx,
			}
		},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithGinEngine(engine *gin.Engine) Option {
	return func(o *_Options) {
		o.engine = engine
	}
}

func WithErrorHeader(errorHeader string) Option {
	return func(o *_Options) {
		o.errorHeader = errorHeader
	}
}

func WithHandleFunc(handleFuncWraps ...HandleFuncWrap) Option {
	return func(o *_Options) {
		o.handleFuncWraps = append(o.handleFuncWraps, handleFuncWraps...)
	}
}

func WithBindReqFunc(bindReqFunc func(req interface{}, ginCtx *gin.Context) error) Option {
	return func(o *_Options) {
		o.bindReqFunc = bindReqFunc
	}
}

func WithCtxFunc(ctxFunc func(ginCtx *gin.Context) interface{}) Option {
	return func(o *_Options) {
		o.ctxFunc = ctxFunc
	}
}

func mongoObjectID() string {
	var bs [12]byte
	binary.BigEndian.PutUint32(bs[0:4], uint32(time.Now().Unix()))
	copy(bs[4:9], processUnique[:])
	putUint24(bs[9:12], atomic.AddUint32(&objectIDCounter, 1))
	return hex.EncodeToString(bs[:])
}

var (
	objectIDCounter = readRandomUint32()
	processUnique   = processUniqueBytes()
)

func processUniqueBytes() [5]byte {
	var b [5]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}
	return b
}

func readRandomUint32() uint32 {
	var b [4]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}
	return (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
