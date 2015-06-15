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
		//if doxContainer.container.ID == "b28a083e1f41c26d86adb09aaaea1ae5789b522ed5e7b729eed40911287656a2" {
		//log.Println(doxContainer.container.Names)
		for _, metric := range config.DockerMetrics {
			go pushContainerStat(influxClient, doxContainer, stat, metric)
		}
		//}
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
	for {
		err := influxClient.Ping()
		if err != nil {
			log.Println("error ping InfluxDB", err)
			routinesDown()
		}
		log.Println("ping ok, influxClient is alive")
		time.Sleep(time.Minute)
	}
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
	go pingInfluxDB(influxClient)
	return influxClient
}
