// File: pkg/taskqueue/taskqueue.go
package taskqueue

import (
	"context"
	"log"
	"sync"
)

// Task adalah interface untuk unit kerja yang bisa dijalankan di background
type Task interface {
	Execute(ctx context.Context) error
}

// TaskFunc adalah adapter untuk menggunakan fungsi biasa sebagai Task
type TaskFunc func(ctx context.Context) error

func (f TaskFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// WorkerPool mengelola antrian tugas dan sekelompok worker
type WorkerPool interface {
	Start()
	Stop()
	Submit(task Task)
}

type workerPool struct {
	numWorkers int
	taskQueue  chan Task
	quit       chan struct{}
	wg         sync.WaitGroup
}

// NewWorkerPool membuat instance baru WorkerPool
func NewWorkerPool(numWorkers int, queueSize int) WorkerPool {
	return &workerPool{
		numWorkers: numWorkers,
		taskQueue:  make(chan Task, queueSize),
		quit:       make(chan struct{}),
	}
}

// Start menjalankan worker-worker di background
func (p *workerPool) Start() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("[TASK-QUEUE] Started %d workers", p.numWorkers)
}

// Stop menghentikan seluruh worker dengan aman (Graceful Shutdown)
func (p *workerPool) Stop() {
	close(p.quit)
	p.wg.Wait()
	close(p.taskQueue)
	log.Println("[TASK-QUEUE] Stopped all workers")
}

// Submit memasukkan tugas baru ke dalam antrian
func (p *workerPool) Submit(task Task) {
	select {
	case p.taskQueue <- task:
		// Tugas berhasil masuk antrian
	case <-p.quit:
		log.Println("[TASK-QUEUE] Warning: Pool is stopping, task rejected")
	default:
		log.Println("[TASK-QUEUE] Warning: Queue is full, task might block or be dropped")
		p.taskQueue <- task // Blocking submit jika antrian penuh
	}
}

func (p *workerPool) worker(id int) {
	defer p.wg.Done()
	log.Printf("[TASK-QUEUE] Worker %d ready", id)

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			// Jalankan tugas
			if err := task.Execute(context.Background()); err != nil {
				log.Printf("[TASK-QUEUE] Worker %d error executing task: %v", id, err)
			}
		case <-p.quit:
			return
		}
	}
}
