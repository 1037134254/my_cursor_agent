package llm

import (
	"strings"
	"sync/atomic"
)

// EndpointScheduler 集群扩容预留：从多个推理 endpoint 中选取下一个。
// 当前实现：轮询（RoundRobin）；后续可替换为加权、健康检查、least-conn 等。
type EndpointScheduler interface {
	Next() string
	Len() int
}

type roundRobinScheduler struct {
	urls []string
	idx  uint64
}

// NewRoundRobinScheduler 从 URL 列表构造调度器；urls 为空时 Next 返回空串。
func NewRoundRobinScheduler(urls []string) EndpointScheduler {
	cp := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			cp = append(cp, u)
		}
	}
	return &roundRobinScheduler{urls: cp}
}

func (r *roundRobinScheduler) Next() string {
	if len(r.urls) == 0 {
		return ""
	}
	i := atomic.AddUint64(&r.idx, 1)
	return r.urls[int(i-1)%len(r.urls)]
}

func (r *roundRobinScheduler) Len() int {
	return len(r.urls)
}
