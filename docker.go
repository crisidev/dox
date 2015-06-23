package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
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

func cleanupDoxContainer(containers []docker.APIContainers, influxClient *influx.Client) {
	for id := range doxContainers {
		if isContainerInAPIContainers(containers, id) == false {
			log.Printf("stopping monitoring for container %s, corresponding series will be deleted in 60 minutes", id)
			go dropContainerSeries(influxClient, stripChars(doxContainers[id].container.Names[0], "/"))
			removeDoxContainer(id)
		}
	}
}

func pullDoxContainers(dockerClient *docker.Client, containers []docker.APIContainers, influxClient *influx.Client) {
	if debugFlag == true {
		log.Printf("running in debug mode, only container %s is monitored", containers[0].ID)
		containers = []docker.APIContainers{containers[0]}
	}
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
			go runInfluxStatPusher(doxContainers[container.ID], influxClient)
		}
	}
}

func updateDoxContainers(dockerClient *docker.Client, influxClient *influx.Client) {
	for {
		containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
		if err != nil {
			log.Fatalln("error reading container list:", err)
		}
		pullDoxContainers(dockerClient, containers, influxClient)
		cleanupDoxContainer(containers, influxClient)
		time.Sleep(time.Second * 10)
	}
}

func runDockerStatCollector(influxClient *influx.Client) {
	log.Println("attaching dockerClient to", config.DockerHost)
	dockerClient, err := docker.NewClient(config.DockerHost)
	if err != nil {
		log.Fatalln("error creating docker client client:", err)
	}
	go updateDoxContainers(dockerClient, influxClient)
}

func dockerIOStatFileToSlice(id string, fileName string) []string {
	cgroupFile := fmt.Sprintf("%s/docker-%s.scope/%s", config.DockerPath["IOStatPath"], id, fileName)
	stats, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		log.Println("error opening stat file", err)
	}
	partials := strings.Split(string(stats), "\n")
	return partials[:len(partials)-1]
}
