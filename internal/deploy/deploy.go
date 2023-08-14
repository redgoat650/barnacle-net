package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
)

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

	list, err := dockerContainerList(node)
	if err != nil {
		return err
	}

	fmt.Println(list)

	for _, cntr := range list {
		fmt.Println("checking image", cntr["Image"])
		if cntr["Image"] == image {
			id := cntr["ID"]
			if id == "" {
				return fmt.Errorf("no ID for container with image %s", cntr["Image"])
			}

			err := dockerContainerStop(id, node)
			if err != nil {
				return fmt.Errorf("stopping container %s from node %s: %s", id, node, err)
			}

			err = dockerContainerRemove(id, node)
			if err != nil {
				return fmt.Errorf("removing container %s from node %s: %s", id, node, err)
			}

			// fmt.Println("IMAGE!", cntr["Names"])

			// err := dockerContainerInspect(node, cntr["ID"])
			// if err != nil {
			// 	return err
			// }

			// break
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
				return fmt.Errorf("removing image %s from node %s: %s", id, node, err)
			}
		}
	}

	barnacleStartCmd := []string{"barnacle", "start", "--server", server}

	err = dockerRun(node, image, barnacleStartCmd...)
	if err != nil {
		return err
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
	allArg    = "--all"
	formatArg = "--format"
	hostArg   = "--host"

	// Static values
	jsonFormatArg = "json"
)

func dockerRun(node, image string, cmd ...string) error {
	// docker run <image> cmd...
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

	args = append(args, runSubCmd, image)
	args = append(args, cmd...)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func dockerContainerStop(id, node string) error {
	// docker --host=<node> container stop <id>
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

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
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

	args = append(args, resource, rmSubCmd, id)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

// func dockerContainerInspect(node, id string) error {
// 	// docker container inspect <id>
// 	var args []string
// 	if node != "" {
// 		args = append(args, hostArgs(node)...)
// 	}

// 	args = append(args, containerSubCmd, inspectSubCmd)
// 	args = append(args, formatArgs()...)

// 	out, err := runDockerCmd(args...)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println("CONTAINER INSPECT:", out)

// 	c := &types.ContainerJSON{}

// 	err = json.Unmarshal([]byte(out), &c)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println(c)

// 	return nil
// }

func dockerContainerList(node string) (ret []formatter.SubHeaderContext, err error) {
	// docker --host <node> container ls --format json
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

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
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

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

// func contextLS() (ret []formatter.ClientContext, err error) {
// 	cmd := exec.Command(dockerCmd, contextSubCmd, listSubCmd, formatArg, jsonFormatArg)

// 	b, err := cmd.Output()
// 	if err != nil {
// 		exitErr := &exec.ExitError{}
// 		if errors.As(err, &exitErr) {
// 			log.Println("STDERR:", exitErr.Stderr)
// 		}

// 		return nil, err
// 	}

// 	for _, s := range strings.Split(strings.TrimSpace(string(b)), "\n") {
// 		clientCtx := &formatter.ClientContext{}

// 		err = json.Unmarshal([]byte(s), &clientCtx)
// 		if err != nil {
// 			return nil, err
// 		}

// 		ret = append(ret, *clientCtx)
// 	}

// 	return ret, nil
// }
