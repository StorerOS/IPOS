// +build !freebsd

package madmin

import (
	diskhw "github.com/shirou/gopsutil/disk"
)

type ServerDiskHwOBDInfo struct {
	Addr       string                           `json:"addr"`
	Usage      []*diskhw.UsageStat              `json:"usages,omitempty"`
	Partitions []diskhw.PartitionStat           `json:"partitions,omitempty"`
	Counters   map[string]diskhw.IOCountersStat `json:"counters,omitempty"`
	Error      string                           `json:"error,omitempty"`
}
