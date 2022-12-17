package fastest

import (
	"google.golang.org/grpc/resolver"
	"testing"
)

func TestBalancer_updateRespTime(t *testing.T) {
	endpoint := `http://localhost:9090`
	// 用中位数
	query := `micro_example_observability_response{kind="server",quantile="0.5"}`

	balancer := &Balancer{
		conns: []*conn{
			{address: resolver.Address{Addr: "127.0.0.1:8081"}},
			{address: resolver.Address{Addr: "127.0.0.1:8082"}},
			{address: resolver.Address{Addr: "127.0.0.1:8083"}},
		},
	}
	balancer.updateRespTime(endpoint, query)
	println(balancer)
}