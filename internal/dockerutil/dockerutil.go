package dockerutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
)

func RunDockerCmd(cmdArgs ...string) (out string, err error) {
	if len(cmdArgs) > 0 && cmdArgs[0] == dockerCmd {
		cmdArgs = cmdArgs[1:]
	}

	log.Println("running command:", dockerCmd, strings.Join(cmdArgs, " "))

	cmd := exec.Command(dockerCmd, cmdArgs...)

	b, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			log.Println("STDERR:", string(exitErr.Stderr))
		}
	}

	return string(b), err
}

const (
	// Command
	dockerCmd = "docker"

	// Subcommands
	containerSubCmd = "container"
	contextSubCmd   = "context"
	execSubCmd      = "exec"
	imageSubCmd     = "image"
	inspectSubCmd   = "inspect"
	logsSubCmd      = "logs"
	listSubCmd      = "ls"
	runSubCmd       = "run"
	stopSubCmd      = "stop"
	rmSubCmd        = "remove"

	// Args
	allArg         = "--all"
	detachedArg    = "--detach"
	deviceArg      = "--device"
	formatArg      = "--format"
	hostArg        = "--host"
	interactiveArg = "--interactive"
	nameArg        = "--name"
	portArg        = "--publish"
	privilegedArg  = "--privileged"
	removeArg      = "--rm"
	restartArg     = "--restart"
	ttyArg         = "--tty"
	volumeMountArg = "--volume"

	// Static values
	jsonFormatArg              = "json"
	UnlessStoppedRestartPolicy = "unless-stopped"
)

type RunOpts struct {
	Name          string
	Detached      bool
	Devices       []string
	Port          []string
	Privileged    bool
	Remove        bool
	RestartPolicy string
	Volumes       []string
}

func argsFromOpts(o RunOpts) (ret []string) {
	if o.Name != "" {
		ret = append(ret, nameArg, o.Name)
	}

	if o.Detached {
		ret = append(ret, detachedArg)
	}

	for _, dev := range o.Devices {
		ret = append(ret, deviceArg, dev)
	}

	for _, ps := range o.Port {
		ret = append(ret, portArg, ps)
	}

	if o.Privileged {
		ret = append(ret, privilegedArg)
	}

	if o.RestartPolicy != "" {
		ret = append(ret, restartArg, o.RestartPolicy)
	}

	if o.Remove {
		ret = append(ret, removeArg)
	}

	for _, vs := range o.Volumes {
		ret = append(ret, volumeMountArg, vs)
	}

	return ret
}

func DockerRun(image, addr string, opts RunOpts, cmd ...string) error {
	// docker run <image> cmd...
	args := hostArgs(addr)

	args = append(args, runSubCmd)

	args = append(args, argsFromOpts(opts)...)

	args = append(args, image)

	args = append(args, cmd...)

	out, err := RunDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func DockerContainerStop(id, addr string) error {
	// docker --host=<node> container stop <id>
	args := hostArgs(addr)

	args = append(args, containerSubCmd, stopSubCmd, id)

	out, err := RunDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func DockerContainerRemove(id, addr string) error {
	return DockerResourceRemove(id, containerSubCmd, addr)
}

func DockerImageRemove(id, addr string) error {
	return DockerResourceRemove(id, imageSubCmd, addr)
}

func DockerResourceRemove(id, resource, addr string) error {
	// docker --host=<node> <resource> remove <id>
	args := hostArgs(addr)

	args = append(args, resource, rmSubCmd, id)

	out, err := RunDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func DockerContainerInspect(id, addr string) ([]types.ContainerJSON, error) {
	// docker container inspect <id> --format json
	args := hostArgs(addr)

	args = append(args, containerSubCmd, inspectSubCmd, id)
	args = append(args, formatArgs()...)

	out, err := RunDockerCmd(args...)
	if err != nil {
		return nil, err
	}

	fmt.Println("CONTAINER INSPECT:", out)

	c := []types.ContainerJSON{}

	err = json.Unmarshal([]byte(out), &c)
	if err != nil {
		return nil, err
	}

	fmt.Println("UNMARSHALLED", c)

	return c, nil
}

func DockerContainerList(addr string) (ret []formatter.SubHeaderContext, err error) {
	// docker --host <node> container ls --format json
	args := hostArgs(addr)

	args = append(args, containerSubCmd, listSubCmd, allArg)
	args = append(args, formatArgs()...)

	out, err := RunDockerCmd(args...)
	if err != nil {
		return nil, err
	}

	for _, s := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(s) == 0 {
			continue
		}

		cntr := &formatter.SubHeaderContext{}

		err = json.Unmarshal([]byte(s), &cntr)
		if err != nil {
			return nil, err
		}

		ret = append(ret, *cntr)
	}

	return ret, nil
}

func DockerImageList(addr string) (ret []formatter.SubHeaderContext, err error) {
	// docker --host <node> image ls --format json
	args := hostArgs(addr)

	args = append(args, imageSubCmd, listSubCmd)
	args = append(args, formatArgs()...)

	out, err := RunDockerCmd(args...)
	if err != nil {
		return nil, err
	}

	for _, s := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(s) == 0 {
			continue
		}

		imgsum := &formatter.SubHeaderContext{}

		err = json.Unmarshal([]byte(s), &imgsum)
		if err != nil {
			return nil, err
		}

		ret = append(ret, *imgsum)
	}

	return ret, nil
}

func DockerLogs(containerName string) (string, error) {
	var args []string

	args = append(args, logsSubCmd)
	args = append(args, containerName)

	return RunDockerCmd(args...)
}

func DockerExec(containerName string, cmd ...string) (string, error) {
	var args []string

	args = append(args, execSubCmd)
	args = append(args, interactiveArg)
	args = append(args, ttyArg)
	args = append(args, containerName)

	args = append(args, cmd...)

	return RunDockerCmd(args...)
}

func formatArgs() []string {
	return []string{
		formatArg,
		jsonFormatArg,
	}
}

func hostArgs(addr string) []string {
	if addr == "" {
		return nil
	}

	return []string{
		hostArg,
		addr,
	}
}
