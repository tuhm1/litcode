package exec

import (
	"archive/tar"
	"context"
	"strings"
	"time"

	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

var cli *client.Client
var ctx = context.TODO()

func init() {
	var e error
	cli, e = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if e != nil {
		panic(e)
	}
	for _, i := range []string{"gcc:latest", "openjdk:8-jdk-alpine", "python:3-alpine"} {
		reader, e := cli.ImagePull(ctx, i, types.ImagePullOptions{})
		if e != nil {
			panic(e)
		}
		defer reader.Close()
		io.ReadAll(reader)
	}
}

type Program struct {
	imageID string
}

func NewCpp(code string) (*Program, string) {
	return newCompiled(
		"gcc:latest",
		code,
		"main.cpp",
		[]string{"g++", "-o", "main", "main.cpp", "-fno-diagnostics-color"},
		[]string{"./main"},
	)
}

func NewJava(code string) (*Program, string) {
	return newCompiled(
		"openjdk:8-jdk-alpine",
		code,
		"Main.java",
		[]string{"javac", "Main.java"},
		[]string{"java", "Main"},
	)
}

func NewPython(code string) *Program {
	return newInterpreted(
		"python:3-alpine",
		code,
		"main.py",
		[]string{"python", "main.py"},
	)
}

type Process struct {
	containerID string
	Stdin       io.Writer
}

func (p *Program) Run(timeLimit time.Duration, memoryLimit int64, stdo io.Writer, stde io.Writer) *Process {
	res, e := cli.ContainerCreate(ctx,
		&container.Config{
			Image:     p.imageID,
			OpenStdin: true,
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:     memoryLimit,
				MemorySwap: memoryLimit,
			},
		}, nil, nil, "")
	if e != nil {
		panic(e)
	}
	containerID := res.ID
	term, e := cli.ContainerAttach(ctx, containerID,
		types.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		})
	if e != nil {
		panic(e)
	}
	go stdcopy.StdCopy(stdo, stde, term.Reader)
	e = cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if e != nil {
		panic(e)
	}
	time.AfterFunc(timeLimit, func() {
		d := 0 * time.Second
		cli.ContainerStop(ctx, containerID, &d)
	})
	return &Process{containerID, term.Conn}
}

func (p *Program) Delete() {
	_, e := cli.ImageRemove(ctx, p.imageID, types.ImageRemoveOptions{})
	if e != nil {
		panic(e)
	}
}

type Stats struct {
	ExitCode int
	Time     time.Duration
	OOM      bool
}

func (p *Process) Wait() {
	cRes, cErr := cli.ContainerWait(ctx, p.containerID, container.WaitConditionNotRunning)
	select {
	case e := <-cErr:
		if e != nil {
			panic(e)
		}
	case <-cRes:
	}
}

func (p *Process) Stats() *Stats {
	stats, e := cli.ContainerInspect(ctx, p.containerID)
	if e != nil {
		panic(e)
	}
	start, e := time.Parse(time.RFC3339Nano, stats.State.StartedAt)
	if e != nil {
		panic(e)
	}
	end, e := time.Parse(time.RFC3339Nano, stats.State.FinishedAt)
	if e != nil {
		panic(e)
	}
	return &Stats{
		ExitCode: stats.State.ExitCode,
		Time:     end.Sub(start),
		OOM:      stats.State.OOMKilled,
	}
}

func (p *Process) Delete() {
	e := cli.ContainerRemove(ctx, p.containerID, types.ContainerRemoveOptions{})
	if e != nil {
		panic(e)
	}
}

func newInterpreted(image string, code string, dest string, runCMD []string) *Program {
	crRes, e := cli.ContainerCreate(ctx,
		&container.Config{
			Image: image,
			Cmd:   runCMD,
		}, nil, nil, nil, "")
	if e != nil {
		panic(e)
	}
	containerID := crRes.ID
	defer cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	copyToContainer(code, dest, containerID)
	if e != nil {
		panic(e)
	}
	cmRes, e := cli.ContainerCommit(ctx, containerID,
		types.ContainerCommitOptions{
			Config: &container.Config{
				Cmd: runCMD,
			},
		})
	if e != nil {
		panic(e)
	}
	return &Program{cmRes.ID}
}

func newCompiled(image string, code string, dest string, compileCMD []string, runCMD []string) (*Program, string) {
	crRes, e := cli.ContainerCreate(ctx,
		&container.Config{
			Image: image,
			Cmd:   compileCMD,
			Tty:   true,
		}, nil, nil, nil, "")
	if e != nil {
		panic(e)
	}
	containerID := crRes.ID
	defer cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	copyToContainer(code, dest, containerID)
	compileErr := compile(containerID)
	if compileErr != "" {
		return nil, compileErr
	}
	cmRes, e := cli.ContainerCommit(ctx, containerID,
		types.ContainerCommitOptions{
			Config: &container.Config{
				Cmd: runCMD,
			},
		})
	if e != nil {
		panic(e)
	}
	return &Program{cmRes.ID}, ""
}

func copyToContainer(code string, dest string, containerID string) {
	r, w := io.Pipe()
	c := make(chan error)
	go func() {
		c <- cli.CopyToContainer(ctx, containerID, ".", r, types.CopyToContainerOptions{})
	}()
	func() {
		defer w.Close()
		tw := tar.NewWriter(w)
		cr := strings.NewReader(code)
		e := tw.WriteHeader(&tar.Header{
			Name: dest,
			Size: cr.Size(),
		})
		if e != nil {
			panic(e)
		}
		_, e = io.Copy(tw, cr)
		if e != nil {
			panic(e)
		}
	}()
	e := <-c
	if e != nil {
		panic(e)
	}
}

func compile(containerID string) string {
	e := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if e != nil {
		panic(e)
	}
	cRes, cErr := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var res container.ContainerWaitOKBody
	select {
	case e := <-cErr:
		if e != nil {
			panic(e)
		}
	case res = <-cRes:
	}
	if res.StatusCode != 0 {
		r, e := cli.ContainerLogs(ctx, containerID,
			types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
			})
		if e != nil {
			panic(e)
		}
		log, e := io.ReadAll(r)
		if e != nil {
			panic(e)
		}
		return string(log)
	}
	return ""
}
