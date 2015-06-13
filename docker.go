package main

import (
	"github.com/fsouza/go-dockerclient"
	"log"
	"time"
)

func isContainerInAPIContainers(slice []docker.APIContainers, id string) bool {
	for _, v := range slice {
		if v.ID == id {
			return true
		}
	}
	return false
}

func cleanupDoxContainer(containers []docker.APIContainers) {
	for id, _ := range doxContainers {
		if isContainerInAPIContainers(containers, id) == false {
			log.Printf("container %s shut down removed from monitoring", id)
			delete(doxContainers, id)
		}
	}
}

func pullDoxContainers(dockerClient *docker.Client, containers []docker.APIContainers) {
	for _, container := range containers {
		if doxContainers[container.ID] == nil {
			doxContainers[container.ID] = &DoxContainer{
				container: container,
				c:         make(chan *docker.Stats),
			}
			log.Println("starting dockerStatPoller for container", container.ID)
			go dockerClient.Stats(docker.StatsOptions{
				ID:    container.ID,
				Stats: doxContainers[container.ID].c,
			})
		}
	}
}

func updateDoxContainers(dockerClient *docker.Client, routine bool) {
	firstRun := true
	for firstRun || routine {
		firstRun = false
		if routine == true {
			time.Sleep(time.Second * 5)
		}
		containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
		if err != nil {
			log.Fatalln("error reading container list:", err)
		}
		// Add new containers and channels
		pullDoxContainers(dockerClient, containers)
		//log.Printf("updated DoxContainers struct. found %d running containers", len(containers))
		// Cleanup of old containers
		cleanupDoxContainer(containers)
	}
}

func runDockerStatCollector() {
	log.Println("attaching dockerClient to", config.DockerHost)
	dockerClient, err := docker.NewClient(config.DockerHost)
	if err != nil {
		log.Fatalln("error creating docker client client:", err)
	}
	updateDoxContainers(dockerClient, false)
	routineUp()
	go updateDoxContainers(dockerClient, true)
}
