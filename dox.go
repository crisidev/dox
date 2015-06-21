package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/fsouza/go-dockerclient"
)

const (
	appName    = "dox"
	appVersion = "2.0"
	appAuthor  = "bigo@crisidev.org"
	appSite    = "https://github.com/crisidev/dox"
)

var (
	versionFlag     bool
	debugFlag       bool
	pidFile         string
	logFile         string
	doxWg           sync.WaitGroup
	routineNum      = 0
	influxWriteChan = make(chan string)
	doxContainers   = make(DoxContainers)
	config          = readDoxConfig("config.json")
)

type DoxConfig struct {
	InfluxHost           string
	InfluxDb             string
	InfluxUser           string
	InfluxPass           string
	DataRetentionMinutes int
	DockerHost           string
	DockerMetrics        []string
	DockerPath           map[string]string
}

type DoxContainer struct {
	statChan  chan *docker.Stats
	container docker.APIContainers
}
type DoxContainers map[string]*DoxContainer

// Init functions
func init() {
	flag.BoolVar(&versionFlag, "version", false, "Print the version number and exit.")
	flag.BoolVar(&versionFlag, "V", false, "Print the version number and exit (shorthand)")

	flag.StringVar(&pidFile, "pidfile", "./dox.pid", "Path of the pid file in daemon mode.")
	flag.StringVar(&pidFile, "p", "./dox.pid", "Path of the pid file in daemon mode (shorthand).")

	flag.StringVar(&logFile, "logfile", "./dox.log", "Path of the log file in daemon mode.")
	flag.StringVar(&logFile, "l", "./dox.log", "Path of the log file in daemon mode (shorthand).")

	flag.BoolVar(&debugFlag, "debug", false, "Run in debug mode only for fist container.")
	flag.BoolVar(&debugFlag, "d", false, "Run in debug mode only for fist container (shothand).")
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

func printDoxInfo() {
	fmt.Printf("%s v%s, docker: %s, influxdb: %s\n", appName, appVersion, config.DockerHost, config.InfluxHost)
}

func sliceContains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

func setupLogging() {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("error opening the log file", err)
	}
	log.SetOutput(file)
}

func sliceIndex(slice []string, element string) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return -1
}

func stripChars(str, delimiter string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(delimiter, r) < 0 {
			return r
		}
		return -1
	}, str)
}

//Signal handling
func routinesDown() {
	copyRoutineNum := routineNum
	for i := 0; i < copyRoutineNum; i++ {
		routineDown()
	}
}

func routineDown() {
	routineNum -= 1
	doxWg.Done()
	log.Printf("routine stopped, %d to go", routineNum)
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
		log.Printf("tearing down %d goroutines, this could take a while...", routineNum)
		routineDown()
	}()
}

func main() {
	flag.Parse()
	if debugFlag == false {
		setupLogging()
	}
	printDoxInfo()
	handleSignals()
	influxClient := getInfluxDBClient()
	runDockerStatCollector(influxClient)
	doxWg.Wait()
	log.Printf("%s stopped", appName)
	os.Exit(0)
}
