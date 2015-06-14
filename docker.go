package main

import (
	"log"
	"time"

	"github.com/fsouza/go-dockerclient"
	influx "github.com/influxdb/influxdb/client"
)

func isContainerInAPIContainers(slice []docker.APIContainers, id string) bool {
	for _, v := range slice {
		if v.ID == id {
			return true
		}
	}
	return false
}

func removeDoxContainer(id string) {
	log.Printf("stopped dockerStatPoller and influxStatPusher for container %s", id)
	delete(doxContainers, id)
}

func cleanupDoxContainer(containers []docker.APIContainers) {
	for id, _ := range doxContainers {
		if isContainerInAPIContainers(containers, id) == false {
			removeDoxContainer(id)
		}
	}
}

func pullDoxContainers(dockerClient *docker.Client, containers []docker.APIContainers, influxClient *influx.Client) {
	for _, container := range containers {
		if doxContainers[container.ID] == nil {
			doxContainers[container.ID] = &DoxContainer{
				container: container,
				statChan:  make(chan *docker.Stats),
			}
			log.Println("starting dockerStatPoller for container", container.ID)
			go dockerClient.Stats(docker.StatsOptions{
				ID:    container.ID,
				Stats: doxContainers[container.ID].statChan,
			})
			routineUp()
			go runInfluxStatPusher(doxContainers[container.ID], influxClient)
		}
	}
}

func updateDoxContainers(dockerClient *docker.Client, influxClient *influx.Client) {
	for daemonFlag {
		containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
		if err != nil {
			log.Fatalln("error reading container list:", err)
		}
		pullDoxContainers(dockerClient, containers, influxClient)
		cleanupDoxContainer(containers)
		time.Sleep(time.Second * 10)
	}
}

func runDockerStatCollector(influxClient *influx.Client) {
	log.Println("attaching dockerClient to", config.DockerHost)
	dockerClient, err := docker.NewClient(config.DockerHost)
	if err != nil {
		log.Fatalln("error creating docker client client:", err)
	}
	routineUp()
	go updateDoxContainers(dockerClient, influxClient)
}
