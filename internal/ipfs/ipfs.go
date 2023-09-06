package ipfs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redgoat650/barnacle-net/internal/dockerutil"
)

const (
	kuboSingletonContainerName = "kubo-bnet"
	kuboImage                  = "ipfs/kubo"
	kuboStartTimeout           = 2 * time.Minute
	kuboStartLogSignal         = "listening on /ip4/0.0.0.0/tcp/8080"

	// IPFS command
	ipfsCmd = "ipfs"

	// Sub-commands
	addSubCmd   = "add"
	peersSubCmd = "peers"
	pinSubCmd   = "pin"
	swarmSubCmd = "swarm"
)

func LaunchSingletonKubo() error {
	containers, err := dockerutil.DockerContainerList("")
	if err != nil {
		return fmt.Errorf("error listing containers: %s", err)
	}

	for _, container := range containers {
		if container["Names"] != kuboSingletonContainerName {
			continue
		}

		if container["State"] != "running" {
			id := container["ID"]
			err := dockerutil.DockerContainerRemove(id, "")
			if err != nil {
				return fmt.Errorf("error removing container id %s: %s", id, err)
			}

			// Break and fallthrough to creating a new container.
			break
		}

		// Container is already running. Nothing left to do.
		return nil
	}

	runOpts := dockerutil.RunOpts{
		Name:          kuboSingletonContainerName,
		Detached:      true,
		Port:          []string{"4001:4001", "4001:4001/udp", "127.0.0.1:8081:8080", "127.0.0.1:5001:5001"},
		RestartPolicy: dockerutil.UnlessStoppedRestartPolicy,
		Volumes:       []string{"kubo-bnet_ipfs_staging:/export", "kubo-bnet_ipfs_data:/data/ipfs"},
	}

	if err := dockerutil.DockerRun(kuboImage, "", runOpts); err != nil {
		return fmt.Errorf("running kubo in docker: %s", err)
	}

	// Watch logs until we see the server listening. https://docs.ipfs.tech/install/run-ipfs-inside-docker/#set-up
	ctx, cancel := context.WithTimeout(context.Background(), kuboStartTimeout)
	defer cancel()

	for {
		time.Sleep(5 * time.Second)

		if ctx.Err() != nil {
			return fmt.Errorf("waiting for kubo container logs to report ready: %s", ctx.Err())
		}

		logOut, err := dockerutil.DockerLogs(kuboSingletonContainerName)
		if err != nil {
			log.Printf("error retrieving docker logs, retrying: %s", err)
		}

		if strings.Contains(logOut, kuboStartLogSignal) {
			// Kubo container ready.
			break
		}
	}

	_, err = dockerutil.DockerExec(kuboSingletonContainerName, ipfsCmd, swarmSubCmd, peersSubCmd)
	if err != nil {
		return fmt.Errorf("error executing swarm peers: %s", err)
	}

	return nil
}

func PinAdd(cids []string) error {
	ipfsPinAddArgs := []string{ipfsCmd, pinSubCmd, addSubCmd}
	_, err := dockerutil.DockerExec(kuboSingletonContainerName, append(ipfsPinAddArgs, cids...)...)
	if err != nil {
		return fmt.Errorf("executing ipfs pin add: %s", err)
	}

	return nil
}
