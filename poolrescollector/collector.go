package poolrescollector

import (
	"fmt"
	"log"
	// "time"

	"github.com/prometheus/client_golang/prometheus"
	// "github.com/shirou/gopsutil/cpu"
	// "github.com/shirou/gopsutil/disk"
	// "github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

type ProcessResourceStats struct {
	PID         int32
	ProcessName string
	Username    string
	CPUPercent  float64
	MemoryRSS   uint64
	MemoryVMS   uint64
	ReadBytes   uint64
	WriteBytes  uint64
}

type PoolResourceCollector struct {
	processCPU    *prometheus.Desc
	processMemRSS *prometheus.Desc
	processMemVMS *prometheus.Desc
	processDiskIO *prometheus.Desc
	errors        prometheus.Counter
}

func NewPoolResourceCollector() *PoolResourceCollector {
	return &PoolResourceCollector{
		processCPU: prometheus.NewDesc(
			"windows_process_cpu_usage_percent",
			"CPU usage percentage per process",
			[]string{"pid", "process_name", "username"},
			nil,
		),
		processMemRSS: prometheus.NewDesc(
			"windows_process_memory_rss_bytes",
			"Process RSS (Resident Set Size) memory usage in bytes",
			[]string{"pid", "process_name", "username"},
			nil,
		),
		processMemVMS: prometheus.NewDesc(
			"windows_process_memory_vms_bytes",
			"Process VMS (Virtual Memory Size) usage in bytes",
			[]string{"pid", "process_name", "username"},
			nil,
		),
		processDiskIO: prometheus.NewDesc(
			"windows_process_disk_io_bytes",
			"Process disk I/O bytes",
			[]string{"pid", "process_name", "username", "operation"},
			nil,
		),
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "windows_poolres_collector_errors_total",
			Help: "Total number of errors encountered while collecting pool resource metrics",
		}),
	}
}

func (collector *PoolResourceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.processCPU
	ch <- collector.processMemRSS
	ch <- collector.processMemVMS
	ch <- collector.processDiskIO
	collector.errors.Describe(ch)
}

func (collector *PoolResourceCollector) Collect(ch chan<- prometheus.Metric) {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting process list: %v\n", err)
		collector.errors.Inc()
		collector.errors.Collect(ch)
		return
	}

	for _, p := range processes {
		stats, err := collector.getProcessStats(p)
		if err != nil {
			continue
		}

		// CPU kullanımı
		ch <- prometheus.MustNewConstMetric(
			collector.processCPU,
			prometheus.GaugeValue,
			stats.CPUPercent,
			fmt.Sprintf("%d", stats.PID),
			stats.ProcessName,
			stats.Username,
		)

		// Memory RSS
		ch <- prometheus.MustNewConstMetric(
			collector.processMemRSS,
			prometheus.GaugeValue,
			float64(stats.MemoryRSS),
			fmt.Sprintf("%d", stats.PID),
			stats.ProcessName,
			stats.Username,
		)

		// Memory VMS
		ch <- prometheus.MustNewConstMetric(
			collector.processMemVMS,
			prometheus.GaugeValue,
			float64(stats.MemoryVMS),
			fmt.Sprintf("%d", stats.PID),
			stats.ProcessName,
			stats.Username,
		)

		// Disk IO - Read
		ch <- prometheus.MustNewConstMetric(
			collector.processDiskIO,
			prometheus.GaugeValue,
			float64(stats.ReadBytes),
			fmt.Sprintf("%d", stats.PID),
			stats.ProcessName,
			stats.Username,
			"read",
		)

		// Disk IO - Write
		ch <- prometheus.MustNewConstMetric(
			collector.processDiskIO,
			prometheus.GaugeValue,
			float64(stats.WriteBytes),
			fmt.Sprintf("%d", stats.PID),
			stats.ProcessName,
			stats.Username,
			"write",
		)
	}
}

func (collector *PoolResourceCollector) getProcessStats(p *process.Process) (*ProcessResourceStats, error) {
	name, err := p.Name()
	if err != nil {
		return nil, err
	}

	username, err := p.Username()
	if err != nil {
		username = "unknown"
	}

	cpuPercent, err := p.CPUPercent()
	if err != nil {
		cpuPercent = 0
	}

	memInfo, err := p.MemoryInfo()
	if err != nil {
		return nil, err
	}

	ioCounters, err := p.IOCounters()
	if err != nil {
		// IO bilgileri alınamazsa 0 olarak ayarla
		ioCounters = &process.IOCountersStat{}
	}

	return &ProcessResourceStats{
		PID:         p.Pid,
		ProcessName: name,
		Username:    username,
		CPUPercent:  cpuPercent,
		MemoryRSS:   memInfo.RSS,
		MemoryVMS:   memInfo.VMS,
		ReadBytes:   ioCounters.ReadBytes,
		WriteBytes:  ioCounters.WriteBytes,
	}, nil
}
