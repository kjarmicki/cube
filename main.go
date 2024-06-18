package main

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"kjarmicki.github.com/cube/manager"
	"kjarmicki.github.com/cube/task"
	"kjarmicki.github.com/cube/worker"
)

func main() {
	host := "localhost"
	mport := 3030
	wport := 3031

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.Api{
		Address: host,
		Port:    wport,
		Worker:  &w,
	}

	go w.RunTasks()
	go wapi.Start()

	workers := []string{fmt.Sprintf("%s:%d", host, wport)}
	m := manager.New(workers)
	mapi := manager.Api{Address: host, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()

	mapi.Start()

	// for i := 0; i < 1; i++ {
	// 	t := task.Task{
	// 		ID:    uuid.New(),
	// 		Name:  fmt.Sprintf("test-container-%d", i),
	// 		State: task.Scheduled,
	// 		Image: "strm/helloworld-http",
	// 	}
	// 	te := task.TaskEvent{
	// 		ID:    uuid.New(),
	// 		State: task.Running,
	// 		Task:  t,
	// 	}
	// 	m.AddTask(te)
	// 	m.SendWork()
	// }

	// go func() {
	// 	for {
	// 		fmt.Println("manager is updating tasks from workers")
	// 		m.UpdateTasks()
	// 		time.Sleep(15 * time.Second)
	// 	}
	// }()

	// for {
	// 	for _, t := range m.TaskDb {
	// 		fmt.Printf("task %s state %d\n", t.ID, t.State)
	// 		time.Sleep(15 * time.Second)
	// 	}
	// }
}
