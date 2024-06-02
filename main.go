package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"kjarmicki.github.com/cube/manager"
	"kjarmicki.github.com/cube/task"
	"kjarmicki.github.com/cube/worker"
)

func main() {
	host := "localhost"
	port := 3030

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	api := worker.Api{
		Address: host,
		Port:    port,
		Worker:  &w,
	}

	go runTasks(&w)
	go api.Start()

	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := manager.New(workers)

	for i := 0; i < 1; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}
		m.AddTask(te)
		m.SendWork()
	}

	go func() {
		for {
			fmt.Println("manager is updating tasks from workers")
			m.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDb {
			fmt.Printf("task %s state %d\n", t.ID, t.State)
			time.Sleep(15 * time.Second)
		}
	}
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("No tasks to process")
		}
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}
