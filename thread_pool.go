package utils

import (
	"fmt"
	"sync"
)

type ThreadPool struct {
	QueueSize int
	PoolSize  int
	Queue     chan Runnable
	wg        *sync.WaitGroup
	destroy   bool
}

type Runnable interface {
	Run()
}

const (
	DefaultThreadQueueSize int = 64
	DefaultThreadPoolSize  int = 8
)

func DefaultThreadPool() *ThreadPool {
	return NewThreadPool(DefaultThreadQueueSize, DefaultThreadPoolSize)
}

func NewThreadPool(QueueSize int, PoolSize int) *ThreadPool {
	pool := &ThreadPool{
		QueueSize: QueueSize,
		PoolSize:  PoolSize,
		Queue:     make(chan Runnable, QueueSize),
		wg:        &sync.WaitGroup{},
		destroy:   false,
	}
	pool.Init()
	return pool
}

func (p *ThreadPool) newThread() {
	go func() {
		for !p.destroy {
			r := <-p.Queue
			r.Run()
			p.wg.Done()
		}
	}()
}

func (p *ThreadPool) Init() {
	for i := 0; i < p.QueueSize; i++ {
		p.newThread()
	}
}

func (p *ThreadPool) Wait() {
	p.wg.Wait()
}

func (p *ThreadPool) Destroy() {
	p.destroy = true
	close(p.Queue)
}

func (p *ThreadPool) Put(r Runnable) error {
	select {
	case p.Queue <- r:
		p.wg.Add(1)
		return nil
	default:
		return fmt.Errorf("thread queue is full")
	}
}
