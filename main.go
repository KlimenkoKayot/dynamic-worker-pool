package main

import (
	"fmt"
	"log"
	"sync"
)

type Task struct {
	id    int
	value interface{}
}

type Result struct {
	id    int
	value interface{}
}

type DynamicWorkerPool struct {
	in  chan Task
	out chan Result

	del chan interface{}

	mu *sync.Mutex
	wg *sync.WaitGroup

	curWorkerCount   int
	availableWorkers int
	maxWorkerCount   int
	lastId           int
}

func (dwp *DynamicWorkerPool) StartWorker() {
	dwp.mu.Lock()
	dwp.curWorkerCount++
	dwp.availableWorkers++
	dwp.mu.Unlock()
	for {
		select {
		case <-dwp.del:
			dwp.mu.Lock()
			dwp.curWorkerCount--
			dwp.availableWorkers--
			dwp.mu.Unlock()
			return
		case task := <-dwp.in:
			dwp.availableWorkers--

			// time.Sleep(1 * time.Second)
			fmt.Printf("[task %d] %s\n", task.id, task.value)

			dwp.availableWorkers++
			dwp.wg.Done()
			fmt.Printf("\t[task %d] Done!\n", task.id)
		}
	}
}

// Создает заданное кол-во воркеров (возможно меньше, если значение больше максимума)
func (dwp *DynamicWorkerPool) CreateWorker(n int) error {
	if n < 0 {
		return fmt.Errorf("cant create negative count of workers!")
	}

	// валидация параметров
	maxCreate := dwp.maxWorkerCount - dwp.curWorkerCount
	n = min(n, maxCreate)
	waitWorkerCount := dwp.curWorkerCount + n

	// запуск (создание) воркеров
	for i := 0; i < n; i++ {
		go dwp.StartWorker()
	}

	// ожидание создания воркеров
	for dwp.curWorkerCount != waitWorkerCount {
		// (если этого не делать, то могут возникнуть ошибки)
	}

	fmt.Printf("\t[%d/%d] Created %d workers...\n", dwp.curWorkerCount, dwp.maxWorkerCount, n)
	return nil
}

func (dwp *DynamicWorkerPool) DeleteWorker(n int) error {
	if n < 0 {
		return fmt.Errorf("cant delete negative count of workers!")
	}

	// валидация параметров
	maxDelete := dwp.curWorkerCount
	n = min(n, maxDelete)
	waitWorkerCount := dwp.curWorkerCount - n

	// остановка (удаление) воркеров
	for i := 0; i < n; i++ {
		dwp.del <- struct{}{}
	}

	// ожидание удаления воркеров
	for dwp.curWorkerCount != waitWorkerCount {
		// (если этого не делать, то могут возникнуть ошибки)
	}

	fmt.Printf("\t[%d/%d] Deleted %d workers...\n", dwp.curWorkerCount, dwp.maxWorkerCount, n)
	return nil
}

func (dwp *DynamicWorkerPool) AddTask(value interface{}) error {

	if dwp.curWorkerCount == 0 {
		return fmt.Errorf("cant add task, because of zero workers!")
	}

	for dwp.availableWorkers == 0 {
		// ожидание доступных воркеров
	}

	dwp.wg.Add(1)
	dwp.in <- Task{
		id:    dwp.lastId,
		value: value,
	}
	dwp.lastId++

	return nil
}

func NewDynamicWorkPool(n int) DynamicWorkerPool {
	return DynamicWorkerPool{
		in:  make(chan Task),
		out: make(chan Result),

		del: make(chan interface{}),

		mu: &sync.Mutex{},
		wg: &sync.WaitGroup{},

		maxWorkerCount: n,
	}
}

// Остановить DWP
func (dwp *DynamicWorkerPool) Stop() {
	dwp.wg.Wait()
	dwp.DeleteWorker(dwp.maxWorkerCount)
	close(dwp.in)
	close(dwp.out)
	close(dwp.del)
}

func main() {
	dwp := NewDynamicWorkPool(100)

	err := dwp.CreateWorker(6)
	if err != nil {
		log.Fatal(err)
	}

	err = dwp.DeleteWorker(3)
	if err != nil {
		log.Fatal(err)
	}

	values := []string{
		"vasiliy",
		"romanov",
		"thx",
		"stepik",
		"golang",
	}

	for _, val := range values {
		err = dwp.AddTask(val)
		if err != nil {
			log.Fatal(err)
		}
	}

	dwp.Stop()
}
