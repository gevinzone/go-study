package web

import (
	"net/http"
	"testing"
)

func BenchmarkStaticRoute(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:   "/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/user/home/bedroom",
		},
	}
	mockHandler := func(ctx *Context) {}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range testRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}

func BenchmarkParamRoute(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/:id",
		},
		{
			method: http.MethodGet,
			path:   "/user/:id/detail",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/123",
		},
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}

func BenchmarkParamRoute2(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/:id/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/:id/blog/:slug",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/789/blog/how-to-write-blog",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}

func BenchmarkRegexRoute(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/:id(^[0-9]+$)",
		},
		{
			method: http.MethodGet,
			path:   "/user/:id(^[0-9]+$)/detail",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/123",
		},
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}
func BenchmarkRegexRoute2(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/:id(^[0-9]+$)/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/:id(^[0-9]+$)/blog/:slug(^[\\w,-]+$)",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/789/blog/how-to-write-blog",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}

func BenchmarkStarRoute(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/*/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/*/*/bedroom",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/789/hotel/bedroom",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}
func BenchmarkStarRoute2(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/*/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/*/blog/*",
		},
	}
	mockHandler := func(ctx *Context) {}
	actualRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/456/detail",
		},
		{
			method: http.MethodGet,
			path:   "/user/789/blog/how-to-write",
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.addRoute(tr.method, tr.path, mockHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tr := range actualRoutes {
			r.findRoute(tr.method, tr.path)
		}
	}
	b.StopTimer()
}
