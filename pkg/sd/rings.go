package sd

import (
	"dynamic-sharding/pkg/consistent"
	"sync"
	"sort"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"context"
	"strings"
)

const numberOfReplicas = 500

var (
	PgwNodeRing    *ConsistentHashNodeRing
	NodeUpdateChan = make(chan []string)
)

// 一致性哈希环,用于管理服务器节点.
type ConsistentHashNodeRing struct {
	ring *consistent.Consistent
	sync.RWMutex
}

func NewConsistentHashNodesRing(nodes []string) *ConsistentHashNodeRing {
	ret := &ConsistentHashNodeRing{ring: consistent.New()}

	ret.SetNumberOfReplicas(numberOfReplicas)
	ret.SetNodes(nodes)
	PgwNodeRing = ret
	return ret
}

func (this *ConsistentHashNodeRing) ReShardRing(nodes []string) {
	this.Lock()
	defer this.Unlock()
	newRing := consistent.New()
	newRing.NumberOfReplicas = numberOfReplicas
	for _, node := range nodes {
		newRing.Add(node)
	}
	this.ring = newRing
}

// 根据pk,获取node节点. chash(pk) -> node
func (this *ConsistentHashNodeRing) GetNode(pk string) (string, error) {
	this.RLock()
	defer this.RUnlock()

	return this.ring.Get(pk)
}

func (this *ConsistentHashNodeRing) SetNodes(nodes []string) {
	for _, node := range nodes {
		this.ring.Add(node)
	}
}

func (this *ConsistentHashNodeRing) SetNumberOfReplicas(num int32) {
	this.ring.NumberOfReplicas = int(num)
}

func StringSliceEqualBCE(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	b = b[:len(a)]
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func RunReshardHashRing(ctx context.Context, logger log.Logger) {
	for {
		select {
		case nodes := <-NodeUpdateChan:


			oldNodes := PgwNodeRing.ring.Members()
			sort.Strings(nodes)
			sort.Strings(oldNodes)
			isEq := StringSliceEqualBCE(nodes, oldNodes)
			if isEq == false {
				level.Info(logger).Log("msg", "RunReshardHashRing_node_update_reshard", "oldnodes", strings.Join(oldNodes, ","), "newnodes", strings.Join(nodes, ","), )
				PgwNodeRing.ReShardRing(nodes)
			} else {
				level.Debug(logger).Log("msg", "RunReshardHashRing_node_same", "nodes", strings.Join(nodes, ","))

			}
		case <-ctx.Done():
			level.Info(logger).Log("msg", "RunReshardHashRingQuit")
			return
		}

	}
}
