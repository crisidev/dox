package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/fsouza/go-dockerclient"
	influx "github.com/influxdb/influxdb/client"
)

func addColumnValueToPoint(column string, value interface{}, index int, point *influx.Series) int {
	point.Columns[index] = column
	point.Points[0][index] = value
	index += 1
	return index
}

func dskIOServiceBytesStatsToPoint(point *influx.Series, id string) *influx.Series {
	stat := dockerIOStatFileToSlice(id, "blkio.throttle.io_service_bytes")
	point = ioStatToPoint(stat, point)
	return point
}

func dskIOServicedStatsToPoint(point *influx.Series, id string) *influx.Series {
	stat := dockerIOStatFileToSlice(id, "blkio.throttle.io_serviced")
	point = ioStatToPoint(stat, point)
	return point
}

func ioStatToPoint(statSlice []string, point *influx.Series) *influx.Series {
	values := [5]int64{0, 0, 0, 0, 0}
	point.Points = make([][]interface{}, 1)
	point.Points[0] = make([]interface{}, 5)
	point.Columns = []string{"read", "write", "sync", "async", "total"}

	for _, v := range statSlice {
		var (
			value  string
			column string
		)
		split := strings.Fields(v)

		if len(split) == 3 {
			column = strings.ToLower(split[1])
			value = split[2]
		} else if len(split) == 2 {
			column = strings.ToLower(split[0])
			value = "0"
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Println("error converting string to int64", err)
		}
		index := sliceIndex(point.Columns, column)
		values[index] += intVal
	}
	for index, v := range values {
		point.Points[0][index] = v
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
