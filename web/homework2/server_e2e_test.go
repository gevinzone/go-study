package web

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

// 这里放着端到端测试的代码

var logMdl = func() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			log.Println(ctx.Req.URL, ctx.Req.Method, ctx.RespStatusCode)
		}
	}
}

var userMdl = func() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			log.Println("handle /user request")
		}
	}
}

var pathMdl = func(path string) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			log.Printf("handle %s request\n", path)
		}
	}
}

func TestServer(t *testing.T) {
	s := NewHTTPServer()
	s.Use(logMdl())
	s.UseV1(http.MethodGet, "/", pathMdl("root"))
	s.UseV1(http.MethodGet, "/home", pathMdl("home"))
	s.Get("/", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, world"))
	})
	s.Get("/home", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, home"))
	})
	s.Get("/user", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, user"))
	}, userMdl())

	s.Post("/form", func(ctx *Context) {
		err := ctx.Req.ParseForm()
		if err != nil {
			fmt.Println(err)
		}
	})

	s.Start(":8081")
}
