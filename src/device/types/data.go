package device

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
)

type Data struct {
	Hostname string `json:"hostname"`
	Temp     float64 `json:"temp"`
	CpuUsage float64 `json:"totalCpuUsage"`
	Ram *mem.VirtualMemoryStat
	Cpu *cpu.InfoStat
	Sensor *sensors.TemperatureStat
}

type ThermalZone struct {
	CurrentTemperature uint32
}

func NewData() (*Data, error) {
	host, _ := os.Hostname()
	percent, _ := cpu.Percent(100 * time.Millisecond, false)
	v, _ := mem.VirtualMemory()

	var cpuInfo cpu.InfoStat
	cpus, err := cpu.Info()
	if err == nil && len(cpus) > 0 {
		cpuInfo = cpus[0]
	}
	
	var sensor sensors.TemperatureStat
	sensors	, err := sensors.SensorsTemperatures()
	if err == nil && len(sensors) > 0{
		sensor = sensors[0]
	} else {
		fmt.Println("Need to run as admin to get sensor access")
	}

	return &Data{
		Hostname: host,
		CpuUsage: math.Round(percent[0]),
		Ram: v,
		Cpu: &cpuInfo,
		Sensor: &sensor,
	}, nil
}

func (d *Data) String() string {
    return fmt.Sprintf(`{Hostname: %s, Temp: %f, TotalCpuUsage: %f}`, d.Hostname, d.Temp, d.CpuUsage)
}

// converts bytes to KB, MB, GB, etc.
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
