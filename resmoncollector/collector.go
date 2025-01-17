package resmoncollector

import (
	"fmt"
	"log"
	"net"
	"syscall"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/process"
)

const (
	TCP_TABLE_OWNER_PID_ALL = 5

	MIB_TCP_STATE_CLOSED     = 1
	MIB_TCP_STATE_LISTEN     = 2
	MIB_TCP_STATE_SYN_SENT   = 3
	MIB_TCP_STATE_SYN_RCVD   = 4
	MIB_TCP_STATE_ESTAB      = 5
	MIB_TCP_STATE_FIN_WAIT1  = 6
	MIB_TCP_STATE_FIN_WAIT2  = 7
	MIB_TCP_STATE_CLOSE_WAIT = 8
	MIB_TCP_STATE_CLOSING    = 9
	MIB_TCP_STATE_LAST_ACK   = 10
	MIB_TCP_STATE_TIME_WAIT  = 11
	MIB_TCP_STATE_DELETE_TCB = 12
)

type MIB_TCPROW_OWNER_PID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

type MIB_TCPTABLE_OWNER_PID struct {
	NumEntries uint32
	Table      [1]MIB_TCPROW_OWNER_PID
}

var (
	modiphlpapi             = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedTcpTable = modiphlpapi.NewProc("GetExtendedTcpTable")
)

type ConnectionStats struct {
	LocalAddr   string
	RemoteAddr  string
	State       string
	ProcessID   uint32
	ProcessName string
	Username    string
}

type NetworkActivityStats struct {
	PID         int32
	ProcessName string
	Address     string
	SendBytes   uint64
	RecvBytes   uint64
	TotalBytes  uint64
}

type CombinedNetworkCollector struct {
	connections     *prometheus.Desc
	errors          prometheus.Counter
	netBytesRecv    *prometheus.Desc
	netBytesSent    *prometheus.Desc
	networkActivity *prometheus.Desc
}

func NewCombinedNetworkCollector() *CombinedNetworkCollector {
	return &CombinedNetworkCollector{
		connections: prometheus.NewDesc(
			"windows_tcp_connections_total",
			"Active TCP connections grouped by state",
			[]string{
				"local_address",
				"remote_address",
				"state",
				"pid",
				"process_name",
				"username",
			},
			nil,
		),
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "windows_network_collector_errors_total",
			Help: "Total number of errors encountered while collecting network metrics",
		}),
		netBytesRecv: prometheus.NewDesc(
			"process_network_receive_bytes",
			"Network bytes received by process",
			[]string{"pid", "process_name", "username"},
			nil,
		),
		netBytesSent: prometheus.NewDesc(
			"process_network_transmit_bytes",
			"Network bytes transmitted by process",
			[]string{"pid", "process_name", "username"},
			nil,
		),
		networkActivity: prometheus.NewDesc(
			"windows_network_activity",
			"Detailed network activity per process and connection",
			[]string{"pid", "process_name", "address", "type"},
			nil,
		),
	}
}

func (collector *CombinedNetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.connections
	ch <- collector.netBytesRecv
	ch <- collector.netBytesSent
	ch <- collector.networkActivity
	collector.errors.Describe(ch)
}

func (collector *CombinedNetworkCollector) Collect(ch chan<- prometheus.Metric) {
	collector.collectTCPConnections(ch)
	collector.collectProcessNetworkStats(ch)
	collector.collectNetworkActivity(ch)
}

func getProcessDetails(pid uint32) (string, string, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return "unknown", "unknown", err
	}

	name, err := p.Name()
	if err != nil {
		name = "unknown"
	}

	username, err := p.Username()
	if err != nil {
		username = "unknown"
	}

	return name, username, nil
}

func (collector *CombinedNetworkCollector) collectTCPConnections(ch chan<- prometheus.Metric) {
	connections, err := getTCPConnections()
	if err != nil {
		log.Printf("Error getting TCP connections: %v\n", err)
		collector.errors.Inc()
		collector.errors.Collect(ch)
		return
	}

	for _, conn := range connections {
		ch <- prometheus.MustNewConstMetric(
			collector.connections,
			prometheus.GaugeValue,
			1.0,
			conn.LocalAddr,
			conn.RemoteAddr,
			conn.State,
			fmt.Sprintf("%d", conn.ProcessID),
			conn.ProcessName,
			conn.Username,
		)
	}
}

func getTCPConnections() ([]ConnectionStats, error) {
	var size uint32
	err := GetExtendedTcpTable(nil, &size, true, syscall.AF_INET, TCP_TABLE_OWNER_PID_ALL, 0)
	if err != nil && err != syscall.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("GetExtendedTcpTable size query failed: %v", err)
	}

	buffer := make([]byte, size)
	err = GetExtendedTcpTable(unsafe.Pointer(&buffer[0]), &size, true, syscall.AF_INET, TCP_TABLE_OWNER_PID_ALL, 0)
	if err != nil {
		return nil, fmt.Errorf("GetExtendedTcpTable failed: %v", err)
	}

	table := (*MIB_TCPTABLE_OWNER_PID)(unsafe.Pointer(&buffer[0]))
	if table.NumEntries == 0 {
		return []ConnectionStats{}, nil
	}

	connections := make([]ConnectionStats, 0, table.NumEntries)
	tablePtr := uintptr(unsafe.Pointer(&buffer[0])) + unsafe.Sizeof(uint32(0))

	for i := uint32(0); i < table.NumEntries; i++ {
		row := (*MIB_TCPROW_OWNER_PID)(unsafe.Pointer(tablePtr + uintptr(i)*unsafe.Sizeof(MIB_TCPROW_OWNER_PID{})))

		localAddr := net.IPv4(byte(row.LocalAddr), byte(row.LocalAddr>>8), byte(row.LocalAddr>>16), byte(row.LocalAddr>>24))
		remoteAddr := net.IPv4(byte(row.RemoteAddr), byte(row.RemoteAddr>>8), byte(row.RemoteAddr>>16), byte(row.RemoteAddr>>24))

		localPort := uint16(row.LocalPort>>8) | uint16(row.LocalPort<<8)
		remotePort := uint16(row.RemotePort>>8) | uint16(row.RemotePort<<8)

		processName, username, err := getProcessDetails(row.OwningPid)
		if err != nil {
			processName = "unknown"
			username = "unknown"
		}

		connections = append(connections, ConnectionStats{
			LocalAddr:   fmt.Sprintf("%s:%d", localAddr.String(), localPort),
			RemoteAddr:  fmt.Sprintf("%s:%d", remoteAddr.String(), remotePort),
			State:       tcpStateName(row.State),
			ProcessID:   row.OwningPid,
			ProcessName: processName,
			Username:    username,
		})
	}

	return connections, nil
}

func (collector *CombinedNetworkCollector) collectProcessNetworkStats(ch chan<- prometheus.Metric) {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting process list: %v\n", err)
		collector.errors.Inc()
		return
	}

	for _, p := range processes {
		pid := p.Pid
		name, err := p.Name()
		if err != nil {
			continue
		}

		username, err := p.Username()
		if err != nil {
			username = "unknown"
		}

		netIO, err := p.IOCounters()
		if err != nil {
			continue
		}

		if netIO.ReadBytes > 0 || netIO.WriteBytes > 0 {
			ch <- prometheus.MustNewConstMetric(
				collector.netBytesRecv,
				prometheus.GaugeValue,
				float64(netIO.ReadBytes),
				fmt.Sprintf("%d", pid),
				name,
				username,
			)

			ch <- prometheus.MustNewConstMetric(
				collector.netBytesSent,
				prometheus.GaugeValue,
				float64(netIO.WriteBytes),
				fmt.Sprintf("%d", pid),
				name,
				username,
			)
		}
	}
}

func (collector *CombinedNetworkCollector) collectNetworkActivity(ch chan<- prometheus.Metric) {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting processes: %v", err)
		collector.errors.Inc()
		return
	}

	for _, p := range processes {
		connections, err := p.Connections()
		if err != nil {
			continue
		}

		name, err := p.Name()
		if err != nil {
			continue
		}

		ioCounters, err := p.IOCounters()
		if err != nil {
			continue
		}

		addressStats := make(map[string]NetworkActivityStats)

		for _, conn := range connections {
			addr := conn.Laddr.IP
			if addr == "" || addr == "0.0.0.0" || addr == "::" {
				addr = conn.Raddr.IP
			}
			if addr == "" {
				continue
			}

			stats := addressStats[addr]
			stats.PID = p.Pid
			stats.ProcessName = name
			stats.Address = addr
			stats.SendBytes = ioCounters.WriteBytes
			stats.RecvBytes = ioCounters.ReadBytes
			stats.TotalBytes = ioCounters.WriteBytes + ioCounters.ReadBytes

			addressStats[addr] = stats
		}

		for _, stats := range addressStats {
			ch <- prometheus.MustNewConstMetric(
				collector.networkActivity,
				prometheus.GaugeValue,
				float64(stats.SendBytes),
				fmt.Sprintf("%d", stats.PID),
				stats.ProcessName,
				stats.Address,
				"send",
			)

			ch <- prometheus.MustNewConstMetric(
				collector.networkActivity,
				prometheus.GaugeValue,
				float64(stats.RecvBytes),
				fmt.Sprintf("%d", stats.PID),
				stats.ProcessName,
				stats.Address,
				"receive",
			)

			ch <- prometheus.MustNewConstMetric(
				collector.networkActivity,
				prometheus.GaugeValue,
				float64(stats.TotalBytes),
				fmt.Sprintf("%d", stats.PID),
				stats.ProcessName,
				stats.Address,
				"total",
			)
		}
	}
}

func GetExtendedTcpTable(pTcpTable unsafe.Pointer, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (errcode error) {
	var bOrderNum uint32
	if bOrder {
		bOrderNum = 1
	}

	r0, _, _ := procGetExtendedTcpTable.Call(
		uintptr(pTcpTable),
		uintptr(unsafe.Pointer(pdwSize)),
		uintptr(bOrderNum),
		uintptr(ulAf),
		uintptr(tableClass),
		uintptr(reserved))

	if r0 != 0 {
		errcode = syscall.Errno(r0)
	}
	return
}

func tcpStateName(state uint32) string {
	states := map[uint32]string{
		MIB_TCP_STATE_CLOSED:     "CLOSED",
		MIB_TCP_STATE_LISTEN:     "LISTEN",
		MIB_TCP_STATE_SYN_SENT:   "SYN_SENT",
		MIB_TCP_STATE_SYN_RCVD:   "SYN_RECEIVED",
		MIB_TCP_STATE_ESTAB:      "ESTABLISHED",
		MIB_TCP_STATE_FIN_WAIT1:  "FIN_WAIT_1",
		MIB_TCP_STATE_FIN_WAIT2:  "FIN_WAIT_2",
		MIB_TCP_STATE_CLOSE_WAIT: "CLOSE_WAIT",
		MIB_TCP_STATE_CLOSING:    "CLOSING",
		MIB_TCP_STATE_LAST_ACK:   "LAST_ACK",
		MIB_TCP_STATE_TIME_WAIT:  "TIME_WAIT",
		MIB_TCP_STATE_DELETE_TCB: "DELETE_TCB",
	}

	if name, ok := states[state]; ok {
		return name
	}
	return "UNKNOWN"
}
