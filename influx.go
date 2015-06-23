package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fsouza/go-dockerclient"
	influx "github.com/influxdb/influxdb/client"
)

func runInfluxStatPusher(doxContainer *DoxContainer, influxClient *influx.Client) {
	log.Printf("starting influxStatPusher for container %s", doxContainer.container.ID)
	for {
		stat := <-doxContainer.statChan
		for _, metric := range config.DockerMetrics {
			go pushContainerStat(influxClient, doxContainer, stat, metric)
		}
	}
}

func pushContainerStat(influxClient *influx.Client, doxContainer *DoxContainer, stat *docker.Stats, metric string) {
	point := &influx.Series{
		Name: fmt.Sprintf("%s.%s", stripChars(doxContainer.container.Names[0], "/"), metric),
	}
	switch {
	case metric == "cpu":
		point = cpuStatsToPoint(point, stat)
	case metric == "mem":
		point = memStatsToPoint(point, stat)
	case metric == "net":
		point = netStatsToPoint(point, stat)
	case metric == "dsk.io_service_bytes":
		point = dskIOServiceBytesStatsToPoint(point, doxContainer.container.ID)
	case metric == "dsk.io_serviced":
		point = dskIOServicedStatsToPoint(point, doxContainer.container.ID)
	default:
	}
	if debugFlag == true {
		log.Println(point)
	}
	go pushContainerPoint(influxClient, point)
}

func dropContainerSeries(influxClient *influx.Client, containerName string) {
	time.Sleep(time.Duration(config.DataRetentionMinutes) * time.Minute)
	log.Println("data retention expiration time reached, expiring dead series from InfluxDB")
	for _, metric := range config.DockerMetrics {
		log.Printf("dropping serie %s.%s from %s", containerName, metric, config.InfluxDb)
		_, err := influxClient.Query(fmt.Sprintf("drop series %s.%s", containerName, metric))
		if err != nil {
			log.Println("error dropping series on InfluxDB:", err)
		}
	}
}

func pushContainerPoint(influxClient *influx.Client, point *influx.Series) {
	err := influxClient.WriteSeriesWithTimePrecision([]*influx.Series{point}, "s")
	if err != nil {
		log.Printf("error sending serie %s to InfluxDB: %s", point.Name, err)
	}
}

func pingInfluxDB(influxClient *influx.Client) {
	retryCount := 5
	for retryCount > 0 {
		err := influxClient.Ping()
		if err != nil {
			log.Printf("error ping InfluxDB, retry in one minute, avaliable retries %d", retryCount)
			log.Println(err)
			retryCount--
		} else {
			if debugFlag == true {
				log.Println("ping ok, influxClient is alive and pushing")
			}
			retryCount = 5
		}
		time.Sleep(time.Minute)
	}
	routineDown()
	log.Fatalln("5 errors pinging InfluxDB, exiting")
}

func getInfluxDBClient() *influx.Client {
	influxConf := influx.ClientConfig{
		Host:     config.InfluxHost,
		Database: config.InfluxDb,
		Username: config.InfluxUser,
		Password: config.InfluxPass,
	}
	log.Printf("attaching influxClient to %s", config.InfluxHost)
	influxClient, err := influx.NewClient(&influxConf)
	if err != nil {
		log.Fatalln("error creating influxdb client for container", err)
	}
	err = influxClient.Ping()
	if err != nil {
		log.Fatalln("error ping InfluxDB, no datatabe to push to", err)
	}
	go pingInfluxDB(influxClient)
	return influxClient
}
