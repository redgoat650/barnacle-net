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

func DeployNodes(img string, nodes ...string) error {
	for _, node := range nodes {
		if err := DeployNode(node, img); err != nil {
			return err
		}
	}

	return nil
}

func DeployNode(node, image string) error {
	fmt.Println("Deploying node", node)

	list, err := dockerContainerList(node)
	if err != nil {
		return err
	}

	fmt.Println(list)

	err = dockerRun(node, "ubuntu")
	if err != nil {
		return err
	}

	list, err = dockerContainerList(node)
	if err != nil {
		return err
	}

	fmt.Println(list)

	return nil
}

const (
	// Command
	dockerCmd = "docker"

	// Subcommands
	containerSubCmd = "container"
	contextSubCmd   = "context"
	listSubCmd      = "ls"
	runSubCmd       = "run"

	// Args
	formatArg = "--format"
	hostArg   = "--host"

	// Static values
	jsonFormatArg = "json"
)

func dockerRun(node, image string) error {
	// docker run <image>
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}

	args = append(args, runSubCmd, image)

	out, err := runDockerCmd(args...)

	fmt.Println("STDOUT:", out)

	return err
}

func dockerContainerList(node string) (ret []formatter.ContainerContext, err error) {
	// docker --host <node> container ls --format json
	var args []string
	if node != "" {
		args = append(args, hostArgs(node)...)
	}
	args = append(args, containerSubCmd, listSubCmd)
	args = append(args, formatArgs()...)

	out, err := runDockerCmd(args...)
	if err != nil {
		return nil, err
	}

	for _, s := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(s) == 0 {
			continue
		}

		clientCtx := &formatter.ContainerContext{}

		err = json.Unmarshal([]byte(s), &clientCtx)
		if err != nil {
			return nil, err
		}

		ret = append(ret, *clientCtx)
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
