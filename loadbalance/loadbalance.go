// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package loadbalance

import (
	"hash/fnv"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ink-yht-code/gint/discovery"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// Select 选择一个实例
	Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error)
	// Name 负载均衡器名称
	Name() string
}

// --- 轮询负载均衡 ---

// RoundRobin 轮询负载均衡
type RoundRobin struct {
	counter uint64
}

// NewRoundRobin 创建轮询负载均衡器
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// Select 选择实例
func (lb *RoundRobin) Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	idx := atomic.AddUint64(&lb.counter, 1) - 1
	return instances[idx%uint64(len(instances))], nil
}

// Name 返回名称
func (lb *RoundRobin) Name() string {
	return "round-robin"
}

// --- 加权轮询负载均衡 ---

// WeightedRoundRobin 加权轮询负载均衡
type WeightedRoundRobin struct {
	mu    sync.Mutex
	state map[string]int // instance ID -> current weight
}

// NewWeightedRoundRobin 创建加权轮询负载均衡器
func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{
		state: make(map[string]int),
	}
}

// Select 选择实例
func (lb *WeightedRoundRobin) Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 清理已经下线的实例状态
	active := make(map[string]struct{}, len(instances))
	for _, inst := range instances {
		active[inst.ID] = struct{}{}
	}
	for id := range lb.state {
		if _, ok := active[id]; !ok {
			delete(lb.state, id)
		}
	}

	var selected *discovery.ServiceInstance
	maxCurrent := 0
	totalWeight := 0

	for _, inst := range instances {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w

		lb.state[inst.ID] += w
		if selected == nil || lb.state[inst.ID] > maxCurrent {
			selected = inst
			maxCurrent = lb.state[inst.ID]
		}
	}

	if selected == nil {
		return instances[0], nil
	}

	lb.state[selected.ID] -= totalWeight
	return selected, nil
}

// Name 返回名称
func (lb *WeightedRoundRobin) Name() string {
	return "weighted-round-robin"
}

// --- 随机负载均衡 ---

// Random 随机负载均衡
type Random struct {
	mu   sync.Mutex
	rand *rand.Rand
}

// NewRandom 创建随机负载均衡器
func NewRandom() *Random {
	return &Random{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select 选择实例
func (lb *Random) Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	lb.mu.Lock()
	idx := lb.rand.Intn(len(instances))
	lb.mu.Unlock()
	return instances[idx], nil
}

// Name 返回名称
func (lb *Random) Name() string {
	return "random"
}

// --- 最少连接负载均衡 ---

// LeastConnections 最少连接负载均衡
type LeastConnections struct {
	mu          sync.Mutex
	connections map[string]int // instance ID -> connection count
}

// NewLeastConnections 创建最少连接负载均衡器
func NewLeastConnections() *LeastConnections {
	return &LeastConnections{
		connections: make(map[string]int),
	}
}

// Select 选择实例
func (lb *LeastConnections) Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 找到连接数最少的实例
	var selected *discovery.ServiceInstance
	minConns := -1

	for _, inst := range instances {
		conns := lb.connections[inst.ID]
		if minConns == -1 || conns < minConns {
			minConns = conns
			selected = inst
		}
	}

	// 增加连接计数
	if selected != nil {
		lb.connections[selected.ID]++
	}

	return selected, nil
}

// Release 释放连接
func (lb *LeastConnections) Release(instance *discovery.ServiceInstance) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.connections[instance.ID] > 0 {
		lb.connections[instance.ID]--
	}
}

// Name 返回名称
func (lb *LeastConnections) Name() string {
	return "least-connections"
}

// --- 一致性哈希负载均衡 ---

// ConsistentHash 一致性哈希负载均衡
type ConsistentHash struct {
	virtualNodes int
	hashFunc     func(string) uint32
	counter      uint64
}

// NewConsistentHash 创建一致性哈希负载均衡器
func NewConsistentHash(virtualNodes ...int) *ConsistentHash {
	vn := 150
	if len(virtualNodes) > 0 && virtualNodes[0] > 0 {
		vn = virtualNodes[0]
	}
	return &ConsistentHash{
		virtualNodes: vn,
		hashFunc:     defaultHashFunc,
	}
}

// Select 选择实例
func (lb *ConsistentHash) Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	idx := atomic.AddUint64(&lb.counter, 1)
	key := "req#" + strconv.FormatUint(idx, 10)
	return lb.SelectByKey(instances, key)
}

// SelectByKey 根据 key 选择实例
func (lb *ConsistentHash) SelectByKey(instances []*discovery.ServiceInstance, key string) (*discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}

	// 构建虚拟节点哈希环
	ring := make([]hashRingEntry, 0, len(instances)*lb.virtualNodes)
	for _, inst := range instances {
		for i := 0; i < lb.virtualNodes; i++ {
			virtualKey := inst.ID + "#" + strconv.Itoa(i)
			hash := lb.hashFunc(virtualKey)
			ring = append(ring, hashRingEntry{
				hash:     hash,
				instance: inst,
			})
		}
	}

	// 排序哈希环
	sort.Slice(ring, func(i, j int) bool {
		return ring[i].hash < ring[j].hash
	})

	// 计算 key 的哈希值
	keyHash := lb.hashFunc(key)

	// 二分查找第一个大于等于 keyHash 的节点
	idx := sort.Search(len(ring), func(i int) bool {
		return ring[i].hash >= keyHash
	})
	if idx == len(ring) {
		idx = 0
	}

	return ring[idx].instance, nil
}

// Name 返回名称
func (lb *ConsistentHash) Name() string {
	return "consistent-hash"
}

type hashRingEntry struct {
	hash     uint32
	instance *discovery.ServiceInstance
}

// defaultHashFunc 默认哈希函数
func defaultHashFunc(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32()
}

// --- 全局负载均衡器 ---

var defaultLoadBalancer LoadBalancer

// SetDefault 设置默认负载均衡器
func SetDefault(lb LoadBalancer) {
	defaultLoadBalancer = lb
}

// Default 获取默认负载均衡器
func Default() LoadBalancer {
	if defaultLoadBalancer == nil {
		defaultLoadBalancer = NewRoundRobin()
	}
	return defaultLoadBalancer
}

// Select 使用默认负载均衡器选择实例
func Select(instances []*discovery.ServiceInstance) (*discovery.ServiceInstance, error) {
	return Default().Select(instances)
}
