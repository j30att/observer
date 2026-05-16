package collectors

import (
	"fmt"
	"time"

	"j30att/observer/internal/agent/repository"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	totalMemoryMetric     = "TotalMemory"
	freeMemoryMetric      = "FreeMemory"
	cpuUtilizationPattern = "CPUutilization%d"
)

type GopsutilCollector struct {
	virtualMemory func() (*mem.VirtualMemoryStat, error)
	cpuPercent    func(interval time.Duration, percpu bool) ([]float64, error)
}

func NewGopsutilCollector() *GopsutilCollector {
	return &GopsutilCollector{
		virtualMemory: mem.VirtualMemory,
		cpuPercent:    cpu.Percent,
	}
}

func (c *GopsutilCollector) Collect(store *repository.MetricsRepository) error {
	memory, err := c.virtualMemory()
	if err != nil {
		return fmt.Errorf("collect virtual memory metrics: %w", err)
	}

	store.SaveGauge(totalMemoryMetric, float64(memory.Total))
	store.SaveGauge(freeMemoryMetric, float64(memory.Free))

	cpuUtilization, err := c.cpuPercent(0, true)
	if err != nil {
		return fmt.Errorf("collect cpu utilization metrics: %w", err)
	}

	for index, value := range cpuUtilization {
		store.SaveGauge(fmt.Sprintf(cpuUtilizationPattern, index+1), value)
	}

	return nil
}
