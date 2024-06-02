package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"kjarmicki.github.com/cube/task"
	"kjarmicki.github.com/cube/worker"
)

type Manager struct {
	Pending       queue.Queue // tasks before submission
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string               // endpoints, as in <hostname>:<port>
	WorkerTaskMap map[string][]uuid.UUID // list of tasks by worker
	TaskWorkerMap map[uuid.UUID]string   // worker by task
	LastWorker    int
}

func New(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) SelectWorker() string {
	var newWorker int
	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
	} else {
		newWorker = 0
	}
	m.LastWorker = newWorker
	return m.Workers[newWorker]
}

func (m *Manager) UpdateTasks() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %s for task updates\n", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error while connecting to %s for task updates\n", worker)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status code from %s (%d)\n", worker, resp.StatusCode)
			continue
		}
		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			log.Printf("Error while decoding response: %v\n", err)
			continue
		}

		for _, t := range tasks {
			log.Printf("Attempting to update task %s\n", t.ID)
			_, ok := m.TaskDb[t.ID]
			if !ok {
				log.Printf("Task with ID %s not found\n", t.ID)
				continue
			}
			m.TaskDb[t.ID].State = t.State
			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()

		// pull a task off the pending queue
		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		t := te.Task

		m.EventDb[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)

		// mark the task as scheduled
		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("Error while marshaling task %s: %v\n", te.Task.ID, err)
			return
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error while connecting to %s: %v\n", url, err)
			m.Pending.Enqueue(te)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&d)
			if err != nil {
				log.Printf("Error while decoding response: %v\n", err)
				return
			}
			log.Printf("Response error (%d): %s\n", resp.StatusCode, e.Message)
			return
		}
		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			log.Printf("Error while decoding response: %v\n", err)
			return
		}
		log.Printf("%#v\n", t)
	} else {
		log.Println("No tasks in the queue")
	}
}
