package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"kjarmicki.github.com/cube/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue              // tasks accepted from the manager, waiting to be run
	Db        map[uuid.UUID]*task.Task // tasks that are currently running
	TaskCount int
}

func (w *Worker) CollectStats() {
	log.Println("[Worker] I will collect stats")
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue() // pull a task off the queue
	if t == nil {
		log.Println("[Worker] no tasks in the queue")
		return task.DockerResult{Error: nil}
	}
	taskQueued := t.(task.Task)

	taskPersisted := w.Db[taskQueued.ID] // if task isn't enqueued, enqueue it
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskPersisted.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("We should not get here")
		}
	} else {
		err := fmt.Errorf("Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}
	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := make([]*task.Task, 0, len(w.Db))
	for _, task := range w.Db {
		tasks = append(tasks, task)
	}
	return tasks
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		log.Printf("[Worker] Error running task %s: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}
	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t
	return result
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("[Worker] Checking status of tasks")
		w.updateTasks()
		log.Println("[Worker] Task updates completed")
		log.Println("[Worker] Sleeping for 15 seconds")
		time.Sleep(time.Second * 15)
	}
}

func (w *Worker) updateTasks() {
	for _, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				log.Printf("[Worker] Error while inspecing task %s: %s\n", t.ID, resp.Error.Error())
			}

			if resp.Container == nil {
				log.Printf("[Worker] No container for running task %s\n", t.ID)
				w.Db[t.ID].State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("[Worker] Container for task %s has exited\n", t.ID)
				w.Db[t.ID].State = task.Failed
			}
			w.Db[t.ID].HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
		}
	}
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("[Worker] Error stopping container %s: %v\n", t.ContainerID, result.Error)
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t
	log.Printf("[Worker] Stopped and removed container %s for task %d\n", t.ContainerID, t.ID)
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("[Worker] Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("[Worker] No tasks to process")
		}
		log.Println("[Worker] Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}
