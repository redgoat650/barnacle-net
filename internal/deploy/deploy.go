package deploy

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

const (
	serverContainerName = "barnacle-server"
)

func DeployServer(image string) error {
	err := cleanupExistingImage(image, "")
	if err != nil {
		return err
	}

	opts := RunOpts{
		Name:     serverContainerName,
		Detached: true,
		Port:     []string{"8080:8080"},
	}

	servCmd := []string{"server", "start"}

	err = dockerRun(image, "", opts, servCmd...)
	if err != nil {
		return err
	}

	// c, err := dockerContainerInspect(serverContainerName, "")
	// if err != nil {
	// 	return "", err
	// }

	// if len(c) != 1 {
	// 	return "", fmt.Errorf("unexpected number of containers returned from inspect: %d", len(c))
	// }

	// ipAddr := c[0].NetworkSettings.IPAddress

	// TODO: Port must be controllable via config
	return nil
}

func DeployNodes(img, server string, nodes ...string) error {
	for _, node := range nodes {
		if err := DeployImageToNode(node, img, server); err != nil {
			return err
		}
	}

	return nil
}

func DeployImageToNode(node, image, server string) error {
	log.Printf("Deploying %s to host %s", image, node)

	err := cleanupExistingImage(image, node)
	if err != nil {
		return err
	}

	barnacleStartCmd := []string{"barnacle", "start", "--server", server}

	opts := RunOpts{
		Name:       "barnacle",
		Detached:   true,
		Devices:    []string{"/dev/gpiomem"},
		Privileged: true,
		Volumes:    []string{"/sys:/sys"},
	}

	err = dockerRun(image, node, opts, barnacleStartCmd...)
	if err != nil {
		return err
	}

	return nil
}

func cleanupExistingImage(image, node string) error {
	var nodeStr string
	if node != "" {
		nodeStr = " from node %s"
	}

	list, err := dockerContainerList(node)
	if err != nil {
		return err
	}

	fmt.Println(list)

	for _, cntr := range list {
		if cntr["Image"] == image {
			id := cntr["ID"]
			if id == "" {
				return fmt.Errorf("no ID for container with image %s", cntr["Image"])
			}

			err := dockerContainerStop(id, node)
			if err != nil {
				return fmt.Errorf("stopping container %s%s: %s", id, nodeStr, err)
			}

			err = dockerContainerRemove(id, node)
			if err != nil {
				return fmt.Errorf("removing container %s%s: %s", id, nodeStr, err)
			}
		}
	}

	imgs, err := dockerImageList(node)
	if err != nil {
		return err
	}

	for _, img := range imgs {
		fmt.Println(img)

		if fmt.Sprintf("%s:%s", img["Repository"], img["Tag"]) == image {
			id := img["ID"]

			err := dockerImageRemove(id, node)
			if err != nil {
				return fmt.Errorf("removing image %s%s: %s", id, nodeStr, err)
			}
		}
	}

	// list, err = dockerContainerList(node)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println(list)

	return nil
}

const (
	// Command
	dockerCmd = "docker"

	// Subcommands
	containerSubCmd = "container"
	contextSubCmd   = "context"
	imageSubCmd     = "image"
	inspectSubCmd   = "inspect"
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
	nameArg        = "--name"
	portArg        = "--publish"
	privilegedArg  = "--privileged"
	removeArg      = "--rm"
	volumeMountArg = "--volume"

	// Static values
	jsonFormatArg = "json"
)

type RunOpts struct {
	Name       string
	Detached   bool
	Devices    []string
	Port       []string
	Privileged bool
	Remove     bool
	Volumes    []string
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

	if o.Remove {
		ret = append(ret, removeArg)
	}

	for _, vs := range o.Volumes {
		ret = append(ret, volumeMountArg, vs)
	}

	return ret
}

func dockerRun(image, node string, opts RunOpts, cmd ...string) error {
	// docker run <image> cmd...
	args := hostArgs(node)

	args = append(args, runSubCmd)

	args = append(args, argsFromOpts(opts)...)

	args = append(args, image)

	args = append(args, cmd...)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func dockerContainerStop(id, node string) error {
	// docker --host=<node> container stop <id>
	args := hostArgs(node)

	args = append(args, containerSubCmd, stopSubCmd, id)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func dockerContainerRemove(id, node string) error {
	return dockerResourceRemove(id, containerSubCmd, node)
}

func dockerImageRemove(id, node string) error {
	return dockerResourceRemove(id, imageSubCmd, node)
}

func dockerResourceRemove(id, resource, node string) error {
	// docker --host=<node> <resource> remove <id>
	args := hostArgs(node)

	args = append(args, resource, rmSubCmd, id)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func dockerContainerInspect(id, node string) ([]types.ContainerJSON, error) {
	// docker container inspect <id> --format json
	args := hostArgs(node)

	args = append(args, containerSubCmd, inspectSubCmd, id)
	args = append(args, formatArgs()...)

	out, err := runDockerCmd(args...)
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

func dockerContainerList(node string) (ret []formatter.SubHeaderContext, err error) {
	// docker --host <node> container ls --format json
	args := hostArgs(node)

	args = append(args, containerSubCmd, listSubCmd, allArg)
	args = append(args, formatArgs()...)

	out, err := runDockerCmd(args...)
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

func dockerImageList(node string) (ret []formatter.SubHeaderContext, err error) {
	// docker --host <node> image ls --format json
	args := hostArgs(node)

	args = append(args, imageSubCmd, listSubCmd)
	args = append(args, formatArgs()...)

	out, err := runDockerCmd(args...)
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

func formatArgs() []string {
	return []string{
		formatArg,
		jsonFormatArg,
	}
}

func hostArgs(node string) []string {
	if node == "" {
		return nil
	}

	return []string{
		hostArg,
		node,
	}
}

func runDockerCmd(cmdArgs ...string) (out string, err error) {
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
