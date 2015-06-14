package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	influx "github.com/influxdb/influxdb/client"
)

func runInfluxStatPusher(doxContainer *DoxContainer, influxClient *influx.Client) {
	log.Printf("starting influxStatPusher for %s", doxContainer.container.ID)
	for daemonFlag {
		stat := <-doxContainer.statChan

		for _, metric := range config.DoxMetrics {
			routineUp()
			go pushContainerStat(influxClient, doxContainer, stat, metric)
		}
	}
}

func pushContainerStat(influxClient *influx.Client, doxContainer *DoxContainer, stat *docker.Stats, metric string) {
	pushContainerPoint(influxClient, prepareContainerPoint(metric, doxContainer, stat))
}

func prepareContainerPoint(pointType string, doxContainer *DoxContainer, stat *docker.Stats) *influx.Series {
	point := &influx.Series{
		Name: fmt.Sprintf("%s.%s", stripChars(doxContainer.container.Names[0], "/"), pointType),
	}
	switch {
	case pointType == "cpu":
		point = cpuStatsToPoint(point, stat)
	case pointType == "mem":
		point = memStatsToPoint(point, stat)
	case pointType == "net":
		point = netStatsToPoint(point, stat)
	case strings.Split(pointType, ".")[0] == "dsk":
		point = dskStatsToPoint(point, stat, pointType)
	default:
	}
	return point
}

func addColumnValueToPoint(column string, value interface{}, index int, point *influx.Series) int {
	point.Columns[index] = column
	point.Points[0][index] = value
	index += 1
	return index
}

func sumDskStats(stat []docker.BlkioStatsEntry) [5]uint64 {
	sum := [5]uint64{0, 0, 0, 0, 0}
	for _, v := range stat {
		switch {
		case v.Op == "Read":
			sum[0] += v.Value
		case v.Op == "Write":
			sum[1] += v.Value
		case v.Op == "Sync":
			sum[2] += v.Value
		case v.Op == "Async":
			sum[3] += v.Value
		case v.Op == "Total":
			sum[4] += v.Value
		default:
		}
	}
	return sum
}

func dskStatsToPoint(point *influx.Series, stat *docker.Stats, pointType string) *influx.Series {
	sum := [5]uint64{}
	point.Points = make([][]interface{}, 1)
	point.Points[0] = make([]interface{}, 5)
	point.Columns = []string{"read", "write", "sync", "async", "total"}
	switch {
	case pointType == "dsk.io_service_bytes_recursive":
		sum = sumDskStats(stat.BlkioStats.IOServiceBytesRecursive)
	case pointType == "dsk.io_serviced_recursive":
		sum = sumDskStats(stat.BlkioStats.IOServicedRecursive)
	case pointType == "dsk.io_queue_recursive":
		sum = sumDskStats(stat.BlkioStats.IOQueueRecursive)
	case pointType == "dsk.io_service_time_recursive":
		sum = sumDskStats(stat.BlkioStats.IOServiceTimeRecursive)
	case pointType == "dsk.io_wait_time_recursive":
		sum = sumDskStats(stat.BlkioStats.IOWaitTimeRecursive)
	case pointType == "dsk.io_merged_recursive":
		sum = sumDskStats(stat.BlkioStats.IOMergedRecursive)
	case pointType == "dsk.io_time_recursive":
		sum = sumDskStats(stat.BlkioStats.IOTimeRecursive)
	case pointType == "dsk.sectors_recursive":
		sum = sumDskStats(stat.BlkioStats.SectorsRecursive)
	default:
	}
	for i, v := range sum {
		point.Points[0][i] = v
	}
	return point
}

func netStatsToPoint(point *influx.Series, stat *docker.Stats) *influx.Series {
	index := 0
	point.Points = make([][]interface{}, 1)
	point.Points[0] = make([]interface{}, 8)
	point.Columns = make([]string, 8)
	index = addColumnValueToPoint("rx_dropped", stat.Network.RxDropped, index, point)
	index = addColumnValueToPoint("rx_bytes", stat.Network.RxBytes, index, point)
	index = addColumnValueToPoint("rx_errors", stat.Network.RxErrors, index, point)
	index = addColumnValueToPoint("rx_packets", stat.Network.RxPackets, index, point)
	index = addColumnValueToPoint("tx_dropped", stat.Network.TxDropped, index, point)
	index = addColumnValueToPoint("tx_bytes", stat.Network.TxBytes, index, point)
	index = addColumnValueToPoint("tx_errors", stat.Network.TxErrors, index, point)
	index = addColumnValueToPoint("tx_packets", stat.Network.TxPackets, index, point)
	return point
}

func memStatsToPoint(point *influx.Series, stat *docker.Stats) *influx.Series {
	index := 0
	point.Points = make([][]interface{}, 1)
	point.Points[0] = make([]interface{}, 33)
	point.Columns = make([]string, 33)
	index = addColumnValueToPoint("total_pgmafault", stat.MemoryStats.Stats.TotalPgmafault, index, point)
	index = addColumnValueToPoint("cache", stat.MemoryStats.Stats.Cache, index, point)
	index = addColumnValueToPoint("mapped_file", stat.MemoryStats.Stats.MappedFile, index, point)
	index = addColumnValueToPoint("total_inactive_file", stat.MemoryStats.Stats.TotalInactiveFile, index, point)
	index = addColumnValueToPoint("pgpgout", stat.MemoryStats.Stats.Pgpgout, index, point)
	index = addColumnValueToPoint("rss", stat.MemoryStats.Stats.Rss, index, point)
	index = addColumnValueToPoint("total_mapped_file", stat.MemoryStats.Stats.TotalMappedFile, index, point)
	index = addColumnValueToPoint("writeback", stat.MemoryStats.Stats.Writeback, index, point)
	index = addColumnValueToPoint("unevictable", stat.MemoryStats.Stats.Unevictable, index, point)
	index = addColumnValueToPoint("pgpgin", stat.MemoryStats.Stats.Pgpgin, index, point)
	index = addColumnValueToPoint("total_unevictable", stat.MemoryStats.Stats.TotalUnevictable, index, point)
	index = addColumnValueToPoint("pgmajfault", stat.MemoryStats.Stats.Pgmajfault, index, point)
	index = addColumnValueToPoint("total_rss", stat.MemoryStats.Stats.TotalRss, index, point)
	index = addColumnValueToPoint("total_rss_huge", stat.MemoryStats.Stats.TotalRssHuge, index, point)
	index = addColumnValueToPoint("total_writeback", stat.MemoryStats.Stats.TotalWriteback, index, point)
	index = addColumnValueToPoint("total_inactive_anon", stat.MemoryStats.Stats.TotalInactiveAnon, index, point)
	index = addColumnValueToPoint("rss_huge", stat.MemoryStats.Stats.RssHuge, index, point)
	index = addColumnValueToPoint("hierarchical_memory_limit", stat.MemoryStats.Stats.HierarchicalMemoryLimit, index, point)
	index = addColumnValueToPoint("total_pgfault", stat.MemoryStats.Stats.TotalPgfault, index, point)
	index = addColumnValueToPoint("total_active_file", stat.MemoryStats.Stats.TotalActiveFile, index, point)
	index = addColumnValueToPoint("active_anon", stat.MemoryStats.Stats.ActiveAnon, index, point)
	index = addColumnValueToPoint("total_active_anon", stat.MemoryStats.Stats.TotalActiveAnon, index, point)
	index = addColumnValueToPoint("total_pgpgout", stat.MemoryStats.Stats.TotalPgpgout, index, point)
	index = addColumnValueToPoint("total_cache", stat.MemoryStats.Stats.TotalCache, index, point)
	index = addColumnValueToPoint("inactive_anon", stat.MemoryStats.Stats.InactiveAnon, index, point)
	index = addColumnValueToPoint("active_file", stat.MemoryStats.Stats.ActiveFile, index, point)
	index = addColumnValueToPoint("pgfault", stat.MemoryStats.Stats.Pgfault, index, point)
	index = addColumnValueToPoint("inactive_file", stat.MemoryStats.Stats.InactiveFile, index, point)
	index = addColumnValueToPoint("total_pgpgin", stat.MemoryStats.Stats.TotalPgpgin, index, point)
	index = addColumnValueToPoint("max_usage", stat.MemoryStats.MaxUsage, index, point)
	index = addColumnValueToPoint("usage", stat.MemoryStats.Usage, index, point)
	index = addColumnValueToPoint("filcnt", stat.MemoryStats.Failcnt, index, point)
	index = addColumnValueToPoint("limit", stat.MemoryStats.Limit, index, point)
	return point
}

func cpuStatsToPoint(point *influx.Series, stat *docker.Stats) *influx.Series {
	index := 0
	point.Points = make([][]interface{}, 1)
	point.Points[0] = make([]interface{}, 10)
	point.Columns = make([]string, 10)
	index = addColumnValueToPoint("cpu_total_usage", stat.CPUStats.CPUUsage.TotalUsage, index, point)

	for i, v := range stat.CPUStats.CPUUsage.PercpuUsage {
		index = addColumnValueToPoint(fmt.Sprintf("cpu_%d_usage", i), v, index, point)
	}
	index = addColumnValueToPoint("cpu_system_usage", stat.CPUStats.SystemCPUUsage, index, point)
	return point
}

func pushContainerPoint(influxClient *influx.Client, point *influx.Series) {
	err := influxClient.WriteSeries([]*influx.Series{point})
	if err != nil {
		log.Println("error sending serie to InfluxDB:", err)
	}
}

func pingInfluxDB(influxClient *influx.Client) {
	for daemonFlag {
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
	routineUp()
	go pingInfluxDB(influxClient)
	return influxClient
}
