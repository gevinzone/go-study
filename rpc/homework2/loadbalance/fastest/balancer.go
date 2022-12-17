package fastest

import (
	"encoding/json"
	"fmt"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework2/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type Balancer struct {
	mutex    sync.RWMutex
	conns    []*conn
	filter   loadbalance.Filter
	lastSync time.Time
	endpoint string
}

func (b *Balancer) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	b.mutex.RLock()
	// 执行过滤，并且挑出响应时间最短的那个节点
	var res *conn
	for _, c := range b.conns {
		if !b.filter(info, c.address) {
			continue
		}
		if res == nil || res.response > c.response {
			res = c
		}

	}
	b.mutex.RUnlock()
	if res == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	return balancer.PickResult{
		SubConn: res.SubConn,
		Done: func(info balancer.DoneInfo) {

		},
	}, nil
}

func (b *Balancer) updateRespTime(endpoint, query string) {
	// 这里很难容错，即如果刷新响应时间失败该怎么办
	httpResp, err := http.Get(fmt.Sprintf("%s/api/v1/query?query=%s", endpoint, query))
	if err != nil {
		// 这里难处理，可以考虑记录错误，然后等下一次
		// 可以考虑中断
		// 也可以重试一定次数之后中断
		log.Fatalln("查询 prometheus 失败", err)
		return
	}
	fmt.Println(httpResp)
	// 解析你从 prometheus 里面拿到的结果
	decoder := json.NewDecoder(httpResp.Body)

	var resp response
	err = decoder.Decode(&resp)
	if err != nil {
		log.Fatalln("反序列化 http 响应失败", err)
		return
	}
	if resp.Status != "success" {
		log.Fatalln("失败的响应", err)
		return
	}
	for _, promRes := range resp.Data.Result {
		address, ok := promRes.Metric["address"]
		if !ok {
			return
		}

		for _, c := range b.conns {
			if c.address.Addr == address {
				ms, err := strconv.ParseInt(promRes.Value[1].(string), 10, 64)
				if err != nil {
					continue
				}
				c.response = time.Duration(ms) * time.Millisecond
			}
		}
	}
}

type Builder struct {
	Filter loadbalance.Filter
	// prometheus 的地址
	Endpoint string
	Query    string
	// 刷新响应时间的间隔
	Interval time.Duration
}

func (b *Builder) Build(info base.PickerBuildInfo) balancer.Picker {
	// 构造链接
	conns := make([]*conn, 0, len(info.ReadySCs))
	for con, val := range info.ReadySCs {
		conns = append(conns, &conn{
			SubConn:  con,
			address:  val.Address,
			response: time.Millisecond * 100,
		})
	}
	// 构造 flt
	flt := b.Filter
	if flt == nil {
		flt = func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	res := &Balancer{
		conns:  conns,
		filter: flt,
	}

	// 在这里考虑定时刷新响应时间，也就是从 prometheus 里面查询
	// 基本上就是启动计时器，调用 updateRespTime
	ch := make(chan struct{}, 1)
	runtime.SetFinalizer(res, func(b *Balancer) {
		ch <- struct{}{}
	})
	go func() {
		ticker := time.NewTicker(b.Interval)
		for {
			select {
			case <-ticker.C:
				// 这里很难容错，即如果刷新响应时间失败该怎么办
				res.updateRespTime(b.Endpoint, b.Query)
			case <-ch:
				return
			}
		}
	}()
	return res
}

func (b *Builder) Name() string {
	return "FASTEST"
}

type conn struct {
	balancer.SubConn
	address resolver.Address
	// 响应时间
	response time.Duration
}

type response struct {
	Status string `json:"status"`
	Data   data   `json:"data"`
}

type data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}
