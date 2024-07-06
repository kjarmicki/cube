package task

import (
	"context"
	"io"
	"log"
	"math"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type State int

const (
	Pending   State = iota // initial state for every task
	Scheduled              // manager has scheduled a task onto a worker
	Running                // worker has successfully started a task
	Completed              // task didn't fail and finished
	Failed
)

type Task struct {
	ID          uuid.UUID
	ContainerID string
	Name        string
	State       State
	// container-specific properties
	Image         string
	Memory        int
	Disk          int
	HostPorts     nat.PortMap
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	// timings
	StartTime  time.Time
	FinishTime time.Time
	// health check
	HealthCheck  string
	RestartCount int
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

func NewConfig(t *Task) Config {
	return Config{
		Name:          t.Name,
		ExposedPorts:  t.ExposedPorts,
		Image:         t.Image,
		Memory:        int64(t.Memory),
		Disk:          int64(t.Disk),
		RestartPolicy: t.RestartPolicy,
	}
}

type DockerInspectResponse struct {
	Error     error
	Container *types.ContainerJSON
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

type Docker struct {
	Client *client.Client
	Config Config
}

func NewDocker(c Config) *Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	return &Docker{
		Client: dc,
		Config: c,
	}
}

// essentially the same as docker run from cli
func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	_, _ = io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}
	r := container.Resources{ // resources required by the container
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}
	cc := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}
	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true, // docker will expose all ports automatically, randomly choosing available ports on host
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	// d.Config.Runtime.ContainerId = resp.ID

	out, err := d.Client.ContainerLogs(
		ctx,
		resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}
	_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Inspect(containerID string) DockerInspectResponse {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()
	resp, err := dc.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("Error inspecting container: %s\n", err.Error())
		return DockerInspectResponse{
			Error: err,
		}
	}
	return DockerInspectResponse{
		Container: &resp,
	}
}

func (d *Docker) Stop(id string) DockerResult {
	ctx := context.Background()
	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{})
	if err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{
		Action: "stop",
		Result: "success",
	}
}
