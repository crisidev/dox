package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	//influx "github.com/influxdb/influxdb/client"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	APP_NAME    = "dox"
	APP_VERSION = "2.0"
	APP_AUTHOR  = "bigo@crisidev.org"
	APP_SITE    = "https://github.com/crisidev/dox"
)

var (
	versionFlag        bool
	daemonFlag         bool
	daemonIntervalFlag time.Duration
	pidFile            string
	config             *DoxConfig
	doxContainers      DoxContainers
	doxWg              sync.WaitGroup
	routineNum         int
)

type DoxConfig struct {
	InfluxHost string
	InfluxDb   string
	InfluxUser string
	InfluxPass string
	DockerHost string
}

type DoxContainer struct {
	c         chan *docker.Stats
	container docker.APIContainers
}
type DoxContainers map[string]*DoxContainer

// Init functions
func init() {
	routineNum = 0
	config = readDoxConfig("config.json")
	doxContainers = make(DoxContainers)
	flag.BoolVar(&versionFlag, "version", false, "Print the version number and exit.")
	flag.BoolVar(&versionFlag, "V", false, "Print the version number and exit (shorthand)")

	flag.BoolVar(&daemonFlag, "daemon", false, "Run in daemon mode.")
	flag.BoolVar(&daemonFlag, "D", false, "Run in daemon mode (shorthand)")

	flag.DurationVar(&daemonIntervalFlag, "interval", time.Second, "Interval between checks in milliseconds in daemon mode.")
	flag.DurationVar(&daemonIntervalFlag, "i", time.Second, "Interval between checks in milliseconds in daemon mode (shorthand).")

	flag.StringVar(&pidFile, "pidfile", "./dox.pid", "Path of the pid file in daemon mode.")
	flag.StringVar(&pidFile, "P", "./dox.pid", "Path of the pid file in daemon mode (shorthand).")
}

// Read config from json file
func readDoxConfig(path string) *DoxConfig {
	configfile, err := os.Open(path)
	if err != nil {
		log.Fatalln("error opening config file:", err)
	}
	confDecoder := json.NewDecoder(configfile)
	configObj := DoxConfig{}
	err = confDecoder.Decode(&configObj)
	if err != nil {
		log.Fatalln("error decoding json config:", err)
	}
	return &configObj
}

// Utils functions
func p(s interface{}) {
	fmt.Println(s)
}

func pc(c DoxContainers) {
	i := 0
	for _, v := range c {
		fmt.Println("--------------------------------")
		fmt.Println(v.container.Names[0])
		i += 1
	}
	fmt.Println(i)
}

func printDoxInfo() {
	fmt.Printf("%s v%s, docker: %s, influxdb: %s\n", APP_NAME, APP_VERSION, config.DockerHost, config.InfluxHost)
}

func sliceContains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

//Signal handling
func routineDown() {
	copyRoutineNum := routineNum
	for i := 0; i < copyRoutineNum; i++ {
		log.Printf("routines stopped, %d to go", routineNum)
		routineNum -= 1
		doxWg.Done()
	}
}

func routineUp() {
	routineNum += 1
	doxWg.Add(1)
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	routineUp()
	go func() {
		_ = <-c
		log.Printf("tearing down %d goroutines. this could take a while...", routineNum)
		routineDown()
	}()
}

func main() {
	flag.Parse()
	printDoxInfo()
	handleSignals()
	runDockerStatCollector()
	runInfluxStatPusher()

	doxWg.Wait()
	os.Exit(0)
}
