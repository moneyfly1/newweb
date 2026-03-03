package worker

import (
	"sync"
)

// Pool Goroutine 池，限制并发数量
type Pool struct {
	workers chan struct{}
	wg      sync.WaitGroup
}

// NewPool 创建指定大小的 Goroutine 池
func NewPool(size int) *Pool {
	return &Pool{
		workers: make(chan struct{}, size),
	}
}

// Submit 提交任务到池中执行
func (p *Pool) Submit(task func()) {
	p.workers <- struct{}{} // 获取 worker
	p.wg.Add(1)

	go func() {
		defer func() {
			<-p.workers // 释放 worker
			p.wg.Done()
		}()
		task()
	}()
}

// Wait 等待所有任务完成
func (p *Pool) Wait() {
	p.wg.Wait()
}

// 全局 worker 池
var (
	defaultPool *Pool
	once        sync.Once
)

// GetDefaultPool 获取默认的 worker 池（100 个并发）
func GetDefaultPool() *Pool {
	once.Do(func() {
		defaultPool = NewPool(100)
	})
	return defaultPool
}
