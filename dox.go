package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/influxdb/influxdb/client"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type DoxConfig struct {
	InfluxHost string
	InfluxDb   string
	InfluxUser string
	InfluxPass string
}

var config = readDoxConfig("config.json")

func main() {
	log.Println("starting dox-agent")
	conn := influxDBConnect()
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: false})
	for {
		for _, container := range containers {
			log.Println("push")
			col1, val1 := getMemoryStats(container.ID)
			col2, val2 := getCpuStats(container.ID)
			col := append(col1, col2...)
			val := append(val1, val2...)
			col = append(col, "container_name")
			val = append(val, container.Names[0])
			InfluxDBWritePoints(conn, col, val)
			time.Sleep(time.Second)

		}
	}
}

// Read config from json file
func readDoxConfig(path string) DoxConfig {
	confFile, err := os.Open(path)
	if err != nil {
		log.Fatalln("error opening config file:", err)
	}
	confDecoder := json.NewDecoder(confFile)
	config := DoxConfig{}
	err = confDecoder.Decode(&config)
	if err != nil {
		log.Fatalln("error decoding json config", err)
	}
	return config
}

// InfluxDB helpers
func influxDBConnect() *client.Client {
	conn, err := client.NewClient(&client.ClientConfig{
		Host:     config.InfluxHost,
		Database: config.InfluxDb,
		Username: config.InfluxUser,
		Password: config.InfluxPass,
	})
	if err != nil {
		log.Fatalln("error:", err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatalln("error:", err)
	}
	log.Printf("ping influxdb %s succeded", config.InfluxHost)
	return conn
}

func InfluxDBWritePoints(conn *client.Client, columns []string, values []interface{}) {
	series := &client.Series{
		Name:    "docker_stats",
		Columns: columns,
		Points: [][]interface{}{
			values,
		},
	}
	err := conn.WriteSeries([]*client.Series{series})
	if err != nil {
		log.Fatalln("error sending serie to influxdb:", err)
	}
}

// Docker helpers
func getCpuStats(id string) ([]string, []interface{}) {
	cgroup := "/sys/fs/cgroup/cpu,cpuacct/system.slice/"
	statfile := "cpuacct.usage"
	columns := []string{"cpu_usage_total"}
	path := fmt.Sprintf("%s/docker-%s.scope/%s", cgroup, id, statfile)
	cputotal, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("error opening cpu stat file %s: %s", path, err)
	}
	partials := []string{strings.Trim(string(cputotal), "\n")}
	statfile = "cpuacct.usage_percpu"
	path = fmt.Sprintf("%s/docker-%s.scope/%s", cgroup, id, statfile)
	cpupartial, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("error opening cpu stat file: %s", err)
	}
	partials = append(partials, strings.Fields(string(cpupartial))...)
	lenght := len(partials) + 1
	values := make([]interface{}, lenght)
	for i, v := range partials {
		intval, _ := strconv.Atoi(v)
		values[i] = intval
		columns = append(columns, fmt.Sprintf("cpu_usage_%d", i))
	}
	return columns, values

}

func getMemoryStats(id string) ([]string, []interface{}) {
	cgroup := "/sys/fs/cgroup/memory/system.slice"
	memory := "memory.stat"
	path := fmt.Sprintf("%s/docker-%s.scope/%s", cgroup, id, memory)
	fd, err := os.Open(path)
	if err != nil {
		log.Printf("error opening cpu stat file: %s", err)
	}
	defer fd.Close()
	scanner := bufio.NewScanner(fd)
	scanner.Split(bufio.ScanLines)
	columns, tmp_values := []string{}, []string{}
	for scanner.Scan() {
		split := strings.Fields(scanner.Text())
		columns = append(columns, split[0])
		tmp_values = append(tmp_values, split[1])
	}
	values := make([]interface{}, len(tmp_values))
	for i, v := range tmp_values {
		values[i], _ = strconv.Atoi(v)
	}

	return columns, values
}
