package deploy

import (
	"fmt"
	"log"
	"strings"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/dockerutil"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/spf13/viper"
)

const (
	serverContainerName = "barnacle-server"
)

func GetValidNodeDeploySettings() (ret []NodeDeploySettings, err error) {
	nodeDeployments := []NodeToDeploy{}

	err = viper.UnmarshalKey(config.DeployNodesCfgPath, &nodeDeployments)
	if err != nil {
		return nil, fmt.Errorf("unable to decode deployment node list: %s", err)
	}

	nodeConfigMap := map[string]message.NodeConfig{}
	err = viper.UnmarshalKey(config.NodesConfigKey, &nodeConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %s", err)
	}

	nameSuperset := make(map[string]struct{})
	for i, nodeDeployCfg := range nodeDeployments {
		if nodeDeployCfg.Name == "" || nodeDeployCfg.Addr == "" {
			return nil, fmt.Errorf("must give node deployment %d a name and address", i)
		}

		if _, ok := nameSuperset[nodeDeployCfg.Name]; ok {
			return nil, fmt.Errorf("name overlaps with existing identifier (no duplicate names allowed): %s", nodeDeployCfg.Name)
		}
		nameSuperset[nodeDeployCfg.Name] = struct{}{}

		thisNodeDeploySettings := NodeDeploySettings{
			NodeToDeploy: nodeDeployCfg,
		}

		if nodeCfg, ok := nodeConfigMap[nodeDeployCfg.Name]; ok {
			if nodeCfg.Orientation != nil {
				orient, ok := config.TranslateOrientation(*nodeCfg.Orientation)
				if !ok {
					return nil, fmt.Errorf("invalid orientation provided for %s: %s", nodeDeployCfg.Name, *nodeCfg.Orientation)
				}

				validatedOrientStr := string(orient)
				nodeCfg.Orientation = &validatedOrientStr
			}

			thisNodeDeploySettings.Config = nodeCfg
		}

		ret = append(ret, thisNodeDeploySettings)
	}

	return ret, nil
}

func DeployServer(image string) error {
	log.Printf("deploying %s as a local server.", image)

	err := cleanupExistingImage(image, "")
	if err != nil {
		return err
	}

	opts := dockerutil.RunOpts{
		Name:          serverContainerName,
		Detached:      true,
		Port:          []string{"8080:8080"},
		RestartPolicy: dockerutil.UnlessStoppedRestartPolicy,
	}

	servCmd := []string{"server", "start"}

	err = dockerutil.DockerRun(image, "", opts, servCmd...)
	if err != nil {
		return err
	}

	// TODO: Port must be controllable via config
	return nil
}

type NodeDeploySettings struct {
	NodeToDeploy
	Config message.NodeConfig `json:"config"`
}

type NodeToDeploy struct {
	Addr string `json:"addr"`
	Name string `json:"name"`
}

func DeployNodes(img, server string, nodes ...NodeDeploySettings) error {
	for _, node := range nodes {
		if err := DeployImageToNode(node, img, server); err != nil {
			return err
		}
	}

	return nil
}

func DeployImageToNode(node NodeDeploySettings, image, server string) error {
	log.Printf("deploying %s to host %s, node will be named %s.", image, node.Addr, node.Name)

	err := cleanupExistingImage(image, node.Addr)
	if err != nil {
		return err
	}

	barnacleStartCmd := []string{
		"barnacle", "start",
		"--server", server,
		"--name", node.Name,
	}

	if node.Config.Orientation != nil {
		barnacleStartCmd = append(barnacleStartCmd, "--orientation", *node.Config.Orientation)
	}

	if node.Config.Labels != nil {
		barnacleStartCmd = append(barnacleStartCmd, "--labels", fmt.Sprintf("%q", strings.Join(node.Config.Labels, ",")))
	}

	opts := dockerutil.RunOpts{
		Name:          "barnacle",
		Detached:      true,
		RestartPolicy: dockerutil.UnlessStoppedRestartPolicy,
		Devices:       []string{"/dev/gpiomem"},
		Privileged:    true,
		Volumes:       []string{"/sys:/sys"},
	}

	err = dockerutil.DockerRun(image, node.Addr, opts, barnacleStartCmd...)
	if err != nil {
		return err
	}

	return nil
}

func cleanupExistingImage(image, nodeAddr string) error {
	var nodeStr string
	if nodeAddr != "" {
		nodeStr = " from node %s"
	}

	list, err := dockerutil.DockerContainerList(nodeAddr)
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

			err := dockerutil.DockerContainerStop(id, nodeAddr)
			if err != nil {
				return fmt.Errorf("stopping container %s%s: %s", id, nodeStr, err)
			}

			err = dockerutil.DockerContainerRemove(id, nodeAddr)
			if err != nil {
				return fmt.Errorf("removing container %s%s: %s", id, nodeStr, err)
			}
		}
	}

	imgs, err := dockerutil.DockerImageList(nodeAddr)
	if err != nil {
		return err
	}

	for _, img := range imgs {
		fmt.Println(img)

		if fmt.Sprintf("%s:%s", img["Repository"], img["Tag"]) == image {
			id := img["ID"]

			err := dockerutil.DockerImageRemove(id, nodeAddr)
			if err != nil {
				return fmt.Errorf("removing image %s%s: %s", id, nodeStr, err)
			}
		}
	}

	return nil
}
