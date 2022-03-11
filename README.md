# 基于gin注重开发效率的http封装包

有了下面一定限制后，我们只需要考虑90%的场景，写一个http api很快捷

- 使用gin的绑定参数api
- 响应体全部为json格式
- 发生了错误，返回http状态码和一个特殊的http响应头部

## 使用gin实现上面限制的经典场景

```
	engine.GET("/hello", func(ctx *gin.Context) {
		// http请求绑定参数到自定义的结构体
		req := new(struct{})
		if err := ctx.ShouldBind(req); err != nil {
			ctx.Header("ERROR", err.Error())
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		// 业务处理完成
		rsp, err := new(struct{}), error(nil)
		code := http.StatusOK
		if err != nil {
			ctx.Header("ERROR", err.Error())
			code = http.StatusInternalServerError
			return
		}
		// 用返回值去响应http
		ctx.JSON(code, rsp)
	})
```

如果我们将http api抽象成调用映射的go方法

```
func (*Example) Hello(req *struct{}, ctx interface{}) (rsp *struct{}, err error) {
	return nil, context.DeadlineExceeded
}
```

- req对应http请求参数，我们最关心的数据，ctx是http上下文
- rsp对应http响应体，err表示有错误
- 这个go方法结构不能说跟grpc很像，只能说一模一样，很符合我们程序员的直觉，所见即所得

但是，gin的api与这个go方法差别很大，在HandlersChain差别也很大，我们更喜欢这种中间处理函数

```
	HandleFunc     func(req interface{}, ctx interface{}) (rsp interface{}, err error)
	HandleFuncWrap func(HandleFunc) HandleFunc
```

- 中间处理函数跟这个go方法很像，能访问这个go方法的参数返回值信息
- 由中间处理函数具体逻辑决定要不要拦截，不需要像gin那样主动调用Abort()

## 解决这些问题

```
func (e *Example) RegisterRoute(server *_http.Server) {
	server.Engine.GET("/hello", server.GinHandlerFunc(func() interface{} { return new(pkg1.HelloRequest) },
		func(req interface{}, ctx interface{}) (interface{}, error) {
			return e.Hello(req.(*pkg1.HelloRequest), ctx.(*_http.DefaultContext))
		}))
}
```

- GinHandlerFunc方法将会根据上面的抽象转成gin的HandlerFunc，还是gin的味道
- 绑定参数的自定义结构体和具体go方法由于使用时才知道，需要额外传入

## 还不够方便

- 每写一个api需要更改两个地方，映射的go方法和注册gin路由函数
- 有新增api时我们希望增加一个go方法就可以了
- 我们借用一下java spring mvc的注解@RequestMapping

```
// @RequestMapping{"method":"GET","path":"/hello"}
func (*Example) Hello(req *struct{}, ctx interface{}) (rsp *struct{}, err error) {
	return nil, context.DeadlineExceeded
}
```

加上@RequestMapping注解的方法就是一个http api，能自动注册到gin路由，但go没有注解

## 我们需要解析注解生成注册gin路由的代码

```
go get -u -t -v github.com/go-productive/_http/_http
_http -inputDir=example
```

- 上面命令会扫描指定目录的带有注释包含@RequestMapping的方法，然后生成注册gin路由代码
- 原理上用到了go编译时的抽象语法树AST

[example](example/example.go)

## 整个仓库代码都非常简单，觉得缺点什么自己fork改造