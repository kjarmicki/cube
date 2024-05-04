package main

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/client"
	"kjarmicki.github.com/cube/task"
)

func main() {
	fmt.Println("starting test container")
	d, dr := createContainer()
	if d == nil {
		os.Exit(1)
	}
	time.Sleep(time.Second * 5)
	stopContainer(d, dr.ContainerId)
}

func createContainer() (*task.Docker, *task.DockerResult) {
	c := task.Config{
		Name:  "test-container-1",
		Image: "postgres:13",
		Env: []string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	}

	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := task.Docker{
		Config: c,
		Client: dc,
	}
	result := d.Run()
	if result.Error != nil {
		return nil, nil
	}

	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)
	return &d, &result
}

func stopContainer(d *task.Docker, id string) *task.DockerResult {
	result := d.Stop(id)
	if result.Error != nil {
		return nil
	}
	fmt.Printf("Container %s has stopped\n", id)
	return &result
}
