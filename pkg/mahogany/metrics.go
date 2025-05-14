package mahogany

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"time"
	"unicode"

	"golang.org/x/sys/unix"

	otel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metric "go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	interval time.Duration

	cpuGauge  metric.Float64Gauge
	memGauge  metric.Float64Gauge
	diskGauge metric.Float64Gauge

	attributes metric.MeasurementOption
}

func NewMetrics(interval time.Duration, attributes map[string]string) (*Metrics, error) {
	meter := otel.Meter("github.com/mpoegel/mahogany")
	cpuUsage, err := meter.Float64Gauge("os.cpu.total",
		metric.WithDescription("percent CPU utilization"),
		metric.WithUnit("percent"))
	if err != nil {
		return nil, err
	}

	memUsage, err := meter.Float64Gauge("os.memory.used",
		metric.WithDescription("percent of memory used"),
		metric.WithUnit("percent"))
	if err != nil {
		return nil, err
	}

	diskUsage, err := meter.Float64Gauge("os.disk.used",
		metric.WithDescription("percent of disk used"),
		metric.WithUnit("percent"))
	if err != nil {
		return nil, err
	}

	metricAttr := make([]attribute.KeyValue, 0)
	for key, val := range attributes {
		metricAttr = append(metricAttr, attribute.String(key, val))
	}

	m := &Metrics{
		interval: interval,

		cpuGauge:   cpuUsage,
		memGauge:   memUsage,
		diskGauge:  diskUsage,
		attributes: metric.WithAttributes(metricAttr...),
	}
	return m, nil
}

func (metrics *Metrics) Collect(ctx context.Context) {
	t := time.NewTicker(metrics.interval)
	lastCpuStat, err := metrics.cpuUsage()
	if err != nil {
		slog.Warn("failed to collect cpu stat", "err", err)
	}

	for {
		select {
		case <-t.C:
			cpuStat, err := metrics.cpuUsage()
			if err != nil {
				slog.Warn("failed to collect cpu stat", "err", err)
			} else if lastCpuStat != nil {
				cpuPercentUtil := float64(cpuStat.TotalBusy()-lastCpuStat.TotalBusy()) * 100.0 /
					float64(cpuStat.TotalBusy()-lastCpuStat.TotalBusy()+cpuStat.Idle-lastCpuStat.Idle)
				metrics.cpuGauge.Record(ctx, cpuPercentUtil, metrics.attributes)
			}
			lastCpuStat = cpuStat

			memStat, err := metrics.memUsage()
			if err != nil {
				slog.Warn("failed to collect memory stat", "err", err)
			} else {
				metrics.memGauge.Record(ctx, float64(memStat.MemTotal-memStat.MemAvailable)/float64(memStat.MemTotal)*100.0, metrics.attributes)
			}

			diskStat, err := metrics.diskUsage()
			if err != nil {
				slog.Warn("failed to collect disk stat", "err", err)
			} else {
				metrics.diskGauge.Record(ctx, float64(diskStat.BlocksTotal-diskStat.BlocksAvailable)/float64(diskStat.BlocksTotal)*100.0, metrics.attributes)
			}
		case <-ctx.Done():
			return
		}
	}
}

type CpuStat struct {
	Num    int
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
}

func (cpu *CpuStat) TotalBusy() uint64 {
	return cpu.User + cpu.Nice + cpu.System
}

func (metrics *Metrics) cpuUsage() (*CpuStat, error) {
	rawUpdate, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}
	stat := &CpuStat{}
	lines := bytes.Split(rawUpdate, []byte("\n"))
	// only look at the aggregate line for now
	parts := bytes.Split(lines[0][5:], []byte(" "))
	stat.User = ParseUint64(parts[0])
	stat.Nice = ParseUint64(parts[1])
	stat.System = ParseUint64(parts[2])
	stat.Idle = ParseUint64(parts[3])
	return stat, nil
}

type MemStat struct {
	MemTotal     uint64
	MemFree      uint64
	MemAvailable uint64
}

func (metrics *Metrics) memUsage() (*MemStat, error) {
	rawUpdate, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	stat := &MemStat{}
	lines := bytes.Split(rawUpdate, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("MemTotal")) {
			stat.MemTotal = FindAndParseUint64(line)
		} else if bytes.HasPrefix(line, []byte("MemFree")) {
			stat.MemFree = FindAndParseUint64(line)
		} else if bytes.HasPrefix(line, []byte("MemAvailable")) {
			stat.MemAvailable = FindAndParseUint64(line)
		}
	}
	return stat, nil
}

type DiskStat struct {
	BlocksTotal     uint64
	BlocksFree      uint64
	BlocksAvailable uint64
	BlockSize       uint64
}

func (metrics *Metrics) diskUsage() (*DiskStat, error) {
	statvfs := unix.Statfs_t{}
	if err := unix.Statfs("/", &statvfs); err != nil {
		return nil, err
	}
	stat := &DiskStat{
		BlocksTotal:     statvfs.Blocks,
		BlocksFree:      statvfs.Bfree,
		BlocksAvailable: statvfs.Bavail,
		BlockSize:       uint64(statvfs.Bsize),
	}
	return stat, nil
}

func FindAndParseUint64(raw []byte) uint64 {
	start := bytes.IndexFunc(raw, unicode.IsDigit)
	end := bytes.LastIndexFunc(raw, unicode.IsDigit)
	return ParseUint64(raw[start : end+1])
}

func ParseUint64(raw []byte) uint64 {
	res := uint64(0)
	for _, n := range raw {
		res = (res * uint64(10)) + uint64(n-48)
	}
	return res
}
