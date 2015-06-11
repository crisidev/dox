package main

import (
	//"bufio"
	//"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	influx "github.com/influxdb/influxdb/client"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	APP_NAME    = "dox"
	APP_VERSION = "0.1"
	APP_AUTHOR  = "bigo@crisidev.org"
	APP_SITE    = "https://github.com/crisidev/dox"
)

var (
	versionFlag        bool
	daemonFlag         bool
	daemonIntervalFlag time.Duration
	pidFile            string
	config             = ReadDoxConfig("config.json")
)

type DoxConfig struct {
	InfluxHost       string
	InfluxDb         string
	InfluxUser       string
	InfluxPass       string
	DockerHost       string
	DockerStatConfig map[string]map[string]string
}

type DoxPoint struct {
	InfluxClient  *influx.Client
	ContainerName string
	MetricName    string
	Columns       []string
	Values        []interface{}
}

func init() {
	flag.BoolVar(&versionFlag, "version", false, "Print the version number and exit.")
	flag.BoolVar(&versionFlag, "V", false, "Print the version number and exit (shorthand)")

	flag.BoolVar(&daemonFlag, "daemon", false, "Run in daemon mode.")
	flag.BoolVar(&daemonFlag, "D", false, "Run in daemon mode (shorthand)")

	flag.DurationVar(&daemonIntervalFlag, "interval", time.Second, "Interval between checks in milliseconds in daemon mode.")
	flag.DurationVar(&daemonIntervalFlag, "i", time.Second, "Interval between checks in milliseconds in daemon mode (shorthand).")

	flag.StringVar(&pidFile, "pidfile", "./dox.pid", "Path of the pid file in daemon mode.")
	flag.StringVar(&pidFile, "P", "./dox.pid", "Path of the pid file in daemon mode (shorthand).")
}

func main() {
	flag.Parse()
	if versionFlag {
		fmt.Printf("%s, version: %s %s\n", APP_NAME, APP_VERSION, APP_SITE)
	} else {
		log.Println("starting dox-agent")
		CreatePid()
		influxClient := InfluxDBConnect()
		dockerClient, _ := docker.NewClient(config.DockerHost)
		containers, _ := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
		firstRun := 0
		for firstRun < 1 || daemonFlag {
			go SendContainersStats(influxClient, containers)
			time.Sleep(daemonIntervalFlag)
			firstRun += 1
		}
	}
}

// Utils Functions
func CreatePid() {
	if pidFile != "" {
		pid := strconv.Itoa(os.Getpid())
		if err := ioutil.WriteFile(pidFile, []byte(pid), 0644); err != nil {
			log.Fatalln("error creating pid file", err)
		}
	}
}

func StripChars(s, delimiter string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(delimiter, r) < 0 {
			return r
		}
		return -1
	}, s)
}

// Read config from json file
func ReadDoxConfig(path string) DoxConfig {
	confFile, err := os.Open(path)
	if err != nil {
		log.Fatalln("error opening config file:", err)
	}
	confDecoder := json.NewDecoder(confFile)
	config := DoxConfig{}
	err = confDecoder.Decode(&config)
	if err != nil {
		log.Fatalln("error decoding json config:", err)
	}
	return config
}

// InfluxDB helpers
func InfluxDBConnect() *influx.Client {
	influxClient, err := influx.NewClient(&influx.ClientConfig{
		Host:     config.InfluxHost,
		Database: config.InfluxDb,
		Username: config.InfluxUser,
		Password: config.InfluxPass,
	})
	if err != nil {
		log.Fatalln("error:", err)
	}
	influxClient.Ping()
	if err := influxClient.Ping(); err != nil {
		log.Fatalln("error:", err)
	}
	log.Printf("ping influxdb %s succeded", config.InfluxHost)
	return influxClient
}

func InfluxDBWritePoint(point DoxPoint) {
	serie := &influx.Series{
		Name:    fmt.Sprintf("%s.%s", point.ContainerName, point.MetricName),
		Columns: point.Columns,
		Points: [][]interface{}{
			point.Values,
		},
	}
	err := point.InfluxClient.WriteSeries([]*influx.Series{serie})
	if err != nil {
		log.Println("error sending serie to influxdb:", err)
	}
}

// Docker helpers
func DockerStats(dockerClient docker.Client, statsCh chan *docker.Stats, containerID string) {
	if err := dockerClient.Stats(docker.StatsOptions{
		ID:    containerID,
		Stats: statsCh,
	}); err != nil {
		log.Panic("unable to read container %s stats: %s", containerID, err)
	}
}

func SendContainersStats(influxClient *influx.Client, containers []docker.APIContainers) {
	for _, container := range containers {
		containerName := StripChars(container.Names[0], "/")
		go SendCpuStats(influxClient, container.ID, containerName)
		go SendMemoryStats(influxClient, container.ID, containerName)
		go SendIOStats(influxClient, container.ID, containerName)
	}
}

func ReadCgroupStatFile(fileType string, containerID string, fileIndex ...int) string {
	fIndex := 0
	if len(fileIndex) > 0 {
		fIndex = fileIndex[0]
	}
	cgroupFile := fmt.Sprintf("%s/docker-%s.scope/%s", config.DockerStatConfig[fileType]["StatPath"],
		containerID, config.DockerStatConfig[fileType][fmt.Sprintf("StatFile%d", fIndex)])
	stats, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		log.Fatalln("error opening %s stat file: %s", fileType, err)
	}
	return string(stats)
}

func SendCpuStats(influxClient *influx.Client, containerID string, containerName string) {
	point := DoxPoint{
		InfluxClient:  influxClient,
		ContainerName: containerName,
		MetricName:    "cpu",
		Columns:       []string{},
	}
	partials := strings.Fields(ReadCgroupStatFile("cpu", containerID))
	point.Values = make([]interface{}, len(partials)+1)
	cpuTotal, i := 0, 0
	for _, v := range partials {
		intVal, err := strconv.Atoi(v)
		if err != nil {
			intVal = 0
			log.Printf("error decoding cpu stat file: %s", err)
		}
		cpuTotal += intVal
		point.Values[i] = intVal
		point.Columns = append(point.Columns, fmt.Sprintf("cpu%d", i))
		i += 1
	}
	point.Values[i] = cpuTotal
	point.Columns = append(point.Columns, "cpu_total")
	InfluxDBWritePoint(point)
}

func StringToBigint(s string) *big.Int {
	val := big.NewInt(0)
	val, err := val.SetString(s, 10)
	if err != true {
		log.Printf("error convering %s to BigInt: %s", s, err)
	}
	return val
}

func SendMemoryStats(influxClient *influx.Client, containerID string, containerName string) {
	point := DoxPoint{
		InfluxClient:  influxClient,
		ContainerName: containerName,
		MetricName:    "mem",
		Columns:       []string{},
	}
	partials := strings.Split(ReadCgroupStatFile("mem", containerID), "\n")
	partials = partials[:len(partials)-1]
	point.Values = make([]interface{}, len(partials))
	i := 0
	for _, v := range partials {
		split := strings.Fields(v)
		point.Columns = append(point.Columns, split[0])
		point.Values[i] = StringToBigint(split[1])
		i += 1
	}
	InfluxDBWritePoint(point)
}

func sliceContains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

func sliceIndex(slice []string, element string) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return -1
}

func SendIOStats(influxClient *influx.Client, containerID string, containerName string) {
	metrics := [][]string{
		[]string{"bytes", ReadCgroupStatFile("disk", containerID, 0)},
		[]string{"iops_queue", ReadCgroupStatFile("disk", containerID, 1)},
	}
	for _, metric := range metrics {
		point := IOStatToPoint(metric[1], DoxPoint{
			InfluxClient:  influxClient,
			ContainerName: containerName,
			MetricName:    fmt.Sprintf("disk.%s", metric[0]),
			Columns:       []string{}})
		go InfluxDBWritePoint(point)
	}
}

func IOStatToPoint(metric string, point DoxPoint) DoxPoint {
	partials := strings.Split(metric, "\n")
	partials = partials[:len(partials)-1]
	values := []*big.Int{}
	for _, v := range partials {
		var (
			column string
			value  string
		)
		split := strings.Fields(v)

		if len(split) == 3 {
			column = split[1]
			value = split[2]
		} else if len(split) == 2 {
			column = split[0]
			value = "0"
		}
		index := sliceIndex(point.Columns, column)

		if index == -1 {
			point.Columns = append(point.Columns, column)
			values = append(values, StringToBigint(value))
		} else {
			values[index] = big.NewInt(0).Add(values[index], StringToBigint(value))
		}

	}
	point.Values = make([]interface{}, 5)
	for i, v := range values {
		point.Values[i] = v
	}
	return point
}
