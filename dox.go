package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/influxdb/influxdb/client"
	"log"
	"os"
	//"time"
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
	//	conn := influxDBConnect()
	getCpuStats("/sys/fs/cgroup/memory/system.slice/docker-4af72814ca8df14cb4aa010b3d64382a12465c445a4eab9bec7278af9b0b3ff2.scope/memory.stat")
	//for {
	//time.Sleep(time.Second)
	//log.Println("push")
	//InfluxDBWritePoints(conn)
	//}
}

// Read config from json file
func readDoxConfig(path string) DoxConfig {
	confFile, err := os.Open(path)
	if err != nil {
		log.Fatalln("error:", err)
	}
	confDecoder := json.NewDecoder(confFile)
	config := DoxConfig{}
	err = confDecoder.Decode(&config)
	if err != nil {
		log.Fatalln("error:", err)
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

func InfluxDBWritePoints(conn *client.Client) {
	series := &client.Series{
		Name:    "docker_stats",
		Columns: []string{"value", "value2", "name"},
		Points: [][]interface{}{
			{1.0, 1.3, "pippo"},
		},
	}
	err := conn.WriteSeries([]*client.Series{series})
	if err != nil {
		log.Fatalln("error:", err)
	}
}

// Docker helpers
func Readln(reader *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = reader.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func readStatFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalln("error:", err)
	}
	reader := bufio.NewReader(file)
	str, e := Readln(reader)
	for e == nil {
		fmt.Println(str)
		str, e = Readln(reader)
	}
}

func getCpuStats(id string) {
	readStatFile(id)
}
