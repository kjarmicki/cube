package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"kjarmicki.github.com/cube/task"
)

type Api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}

type MemStats struct {
	MemTotal     int `json:"mem_total"`
	MemFree      int `json:"mem_free"`
	MemAvailable int `json:"mem_available"`
}

type DiskStats struct {
	All        int64 `json:"all"`
	Used       int64 `json:"used"`
	Free       int64 `json:"free"`
	FreeInodes int   `json:"freeInodes"`
}

type CpuStats struct {
	ID        string `json:"id"`
	User      int    `json:"user"`
	Nice      int    `json:"nice"`
	System    int    `json:"system"`
	Idle      int    `json:"idle"`
	Iowait    int    `json:"iowait"`
	Irq       int    `json:"irq"`
	Softirq   int    `json:"softirq"`
	Steal     int    `json:"steal"`
	Guest     int    `json:"guest"`
	GuestNice int    `json:"guest_nice"`
}

type LoadStats struct {
	Last1Min       float64 `json:"last1min"`
	Last5Min       float64 `json:"last5min"`
	Last15Min      float64 `json:"last15min"`
	ProcessRunning int     `json:"process_running"`
	ProcessTotal   int     `json:"process_total"`
	LastPID        int     `json:"last_pid"`
}

type Stats struct {
	MemStats  MemStats  `json:"MemStats"`
	DiskStats DiskStats `json:"DiskStats"`
	CpuStats  CpuStats  `json:"CpuStats"`
	LoadStats LoadStats `json:"LoadStats"`
	TaskCount int       `json:"TaskCount"`
}

type ErrResponse struct {
	Message string
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", a.StopTaskHandler)
		})
	})
	a.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", a.FakeStats)
	})
}

func (a *Api) Start() {
	a.initRouter()
	_ = http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Print(msg)
		w.WriteHeader(400)
		e := ErrResponse{
			Message: msg,
		}
		_ = json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %s\n", te.Task.ID)
	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	_ = json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("No taskID passed in the request\n")
		w.WriteHeader(400)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		log.Printf("TaskID passed in the request looks invalid\n")
		w.WriteHeader(400)
		return
	}
	taskToStop, ok := a.Worker.Db[tID]
	if !ok {
		log.Printf("Task not found by ID\n")
		w.WriteHeader(400)
		return
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	log.Printf("Added task %s to stop container %s", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(204)
}

func (a *Api) FakeStats(w http.ResponseWriter, r *http.Request) {
	stats := Stats{
		MemStats: MemStats{
			MemTotal:     32488372,
			MemFree:      14399056,
			MemAvailable: 23306576,
		},
		DiskStats: DiskStats{
			All:        1006660349952,
			Used:       39346565120,
			Free:       967313784832,
			FreeInodes: 61645909,
		},
		CpuStats: CpuStats{
			ID:        "cpu",
			User:      4819423,
			Nice:      701,
			System:    2140212,
			Idle:      502094668,
			Iowait:    14448,
			Irq:       561115,
			Softirq:   178454,
			Steal:     0,
			Guest:     0,
			GuestNice: 0,
		},
		LoadStats: LoadStats{
			Last1Min:       0.78,
			Last5Min:       0.55,
			Last15Min:      0.43,
			ProcessRunning: 2,
			ProcessTotal:   2336,
			LastPID:        581117,
		},
		TaskCount: 0,
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	_ = json.NewEncoder(w).Encode(stats)
}
