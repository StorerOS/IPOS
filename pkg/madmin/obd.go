package madmin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	nethw "github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"

	"github.com/storeros/ipos/pkg/disk"
	"github.com/storeros/ipos/pkg/net"
)

type OBDInfo struct {
	TimeStamp time.Time   `json:"timestamp,omitempty"`
	Error     string      `json:"error,omitempty"`
	Perf      PerfOBDInfo `json:"perf,omitempty"`
	IPOS      IPOSOBDInfo `json:"ipos,omitempty"`
	Sys       SysOBDInfo  `json:"sys,omitempty"`
}

type SysOBDInfo struct {
	CPUInfo    []ServerCPUOBDInfo    `json:"cpus,omitempty"`
	DiskHwInfo []ServerDiskHwOBDInfo `json:"disks,omitempty"`
	OsInfo     []ServerOsOBDInfo     `json:"osinfos,omitempty"`
	MemInfo    []ServerMemOBDInfo    `json:"meminfos,omitempty"`
	ProcInfo   []ServerProcOBDInfo   `json:"procinfos,omitempty"`
	Error      string                `json:"error,omitempty"`
}

type ServerProcOBDInfo struct {
	Addr      string          `json:"addr"`
	Processes []SysOBDProcess `json:"processes,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type SysOBDProcess struct {
	Pid            int32                       `json:"pid"`
	Background     bool                        `json:"background,omitempty"`
	CPUPercent     float64                     `json:"cpupercent,omitempty"`
	Children       []int32                     `json:"children,omitempty"`
	CmdLine        string                      `json:"cmd,omitempty"`
	Connections    []nethw.ConnectionStat      `json:"connections,omitempty"`
	CreateTime     int64                       `json:"createtime,omitempty"`
	Cwd            string                      `json:"cwd,omitempty"`
	Exe            string                      `json:"exe,omitempty"`
	Gids           []int32                     `json:"gids,omitempty"`
	IOCounters     *process.IOCountersStat     `json:"iocounters,omitempty"`
	IsRunning      bool                        `json:"isrunning,omitempty"`
	MemInfo        *process.MemoryInfoStat     `json:"meminfo,omitempty"`
	MemMaps        *[]process.MemoryMapsStat   `json:"memmaps,omitempty"`
	MemPercent     float32                     `json:"mempercent,omitempty"`
	Name           string                      `json:"name,omitempty"`
	NetIOCounters  []nethw.IOCountersStat      `json:"netiocounters,omitempty"`
	Nice           int32                       `json:"nice,omitempty"`
	NumCtxSwitches *process.NumCtxSwitchesStat `json:"numctxswitches,omitempty"`
	NumFds         int32                       `json:"numfds,omitempty"`
	NumThreads     int32                       `json:"numthreads,omitempty"`
	OpenFiles      []process.OpenFilesStat     `json:"openfiles,omitempty"`
	PageFaults     *process.PageFaultsStat     `json:"pagefaults,omitempty"`
	Parent         int32                       `json:"parent,omitempty"`
	Ppid           int32                       `json:"ppid,omitempty"`
	Rlimit         []process.RlimitStat        `json:"rlimit,omitempty"`
	Status         string                      `json:"status,omitempty"`
	Tgid           int32                       `json:"tgid,omitempty"`
	Threads        map[int32]*cpu.TimesStat    `json:"threadstats,omitempty"`
	Times          *cpu.TimesStat              `json:"cputimes,omitempty"`
	Uids           []int32                     `json:"uidsomitempty"`
	Username       string                      `json:"username,omitempty"`
}

type ServerMemOBDInfo struct {
	Addr       string                 `json:"addr"`
	SwapMem    *mem.SwapMemoryStat    `json:"swap,omitempty"`
	VirtualMem *mem.VirtualMemoryStat `json:"virtualmem,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type ServerOsOBDInfo struct {
	Addr    string                 `json:"addr"`
	Info    *host.InfoStat         `json:"info,omitempty"`
	Sensors []host.TemperatureStat `json:"sensors,omitempty"`
	Users   []host.UserStat        `json:"users,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type ServerCPUOBDInfo struct {
	Addr     string          `json:"addr"`
	CPUStat  []cpu.InfoStat  `json:"cpu,omitempty"`
	TimeStat []cpu.TimesStat `json:"time,omitempty"`
	Error    string          `json:"error,omitempty"`
}

type IPOSOBDInfo struct {
	Info   InfoMessage `json:"info,omitempty"`
	Config interface{} `json:"config,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type PerfOBDInfo struct {
	DriveInfo   []ServerDrivesOBDInfo `json:"drives,omitempty"`
	Net         []ServerNetOBDInfo    `json:"net,omitempty"`
	NetParallel ServerNetOBDInfo      `json:"net_parallel,omitempty"`
	Error       string                `json:"error,omitempty"`
}

type ServerDrivesOBDInfo struct {
	Addr     string         `json:"addr"`
	Serial   []DriveOBDInfo `json:"serial,omitempty"`
	Parallel []DriveOBDInfo `json:"parallel,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type DriveOBDInfo struct {
	Path       string          `json:"endpoint"`
	Latency    disk.Latency    `json:"latency,omitempty"`
	Throughput disk.Throughput `json:"throughput,omitempty"`
	Error      string          `json:"error,omitempty"`
}

type ServerNetOBDInfo struct {
	Addr  string       `json:"addr"`
	Net   []NetOBDInfo `json:"net,omitempty"`
	Error string       `json:"error,omitempty"`
}

type NetOBDInfo struct {
	Addr       string         `json:"remote"`
	Latency    net.Latency    `json:"latency,omitempty"`
	Throughput net.Throughput `json:"throughput,omitempty"`
	Error      string         `json:"error,omitempty"`
}

type OBDDataType string

const (
	OBDDataTypePerfDrive  OBDDataType = "perfdrive"
	OBDDataTypePerfNet    OBDDataType = "perfnet"
	OBDDataTypeIPOSInfo   OBDDataType = "iposinfo"
	OBDDataTypeIPOSConfig OBDDataType = "iposconfig"
	OBDDataTypeSysCPU     OBDDataType = "syscpu"
	OBDDataTypeSysDiskHw  OBDDataType = "sysdiskhw"
	OBDDataTypeSysDocker  OBDDataType = "sysdocker"
	OBDDataTypeSysOsInfo  OBDDataType = "sysosinfo"
	OBDDataTypeSysLoad    OBDDataType = "sysload"
	OBDDataTypeSysMem     OBDDataType = "sysmem"
	OBDDataTypeSysNet     OBDDataType = "sysnet"
	OBDDataTypeSysProcess OBDDataType = "sysprocess"
)

var OBDDataTypesMap = map[string]OBDDataType{
	"perfdrive":  OBDDataTypePerfDrive,
	"perfnet":    OBDDataTypePerfNet,
	"iposinfo":   OBDDataTypeIPOSInfo,
	"iposconfig": OBDDataTypeIPOSConfig,
	"syscpu":     OBDDataTypeSysCPU,
	"sysdiskhw":  OBDDataTypeSysDiskHw,
	"sysdocker":  OBDDataTypeSysDocker,
	"sysosinfo":  OBDDataTypeSysOsInfo,
	"sysload":    OBDDataTypeSysLoad,
	"sysmem":     OBDDataTypeSysMem,
	"sysnet":     OBDDataTypeSysNet,
	"sysprocess": OBDDataTypeSysProcess,
}

var OBDDataTypesList = []OBDDataType{
	OBDDataTypePerfDrive,
	OBDDataTypePerfNet,
	OBDDataTypeIPOSInfo,
	OBDDataTypeIPOSConfig,
	OBDDataTypeSysCPU,
	OBDDataTypeSysDiskHw,
	OBDDataTypeSysDocker,
	OBDDataTypeSysOsInfo,
	OBDDataTypeSysLoad,
	OBDDataTypeSysMem,
	OBDDataTypeSysNet,
	OBDDataTypeSysProcess,
}

func (adm *AdminClient) ServerOBDInfo(ctx context.Context, obdDataTypes []OBDDataType, deadline time.Duration) <-chan OBDInfo {
	respChan := make(chan OBDInfo)
	go func() {
		v := url.Values{}

		v.Set("deadline",
			deadline.Truncate(1*time.Second).String())

		for _, d := range OBDDataTypesList {
			v.Set(string(d), "false")
		}

		for _, d := range obdDataTypes {
			v.Set(string(d), "true")
		}
		var OBDInfoMessage OBDInfo
		OBDInfoMessage.TimeStamp = time.Now()

		if v.Get(string(OBDDataTypeIPOSInfo)) == "true" {
			info, err := adm.ServerInfo(ctx)
			if err != nil {
				respChan <- OBDInfo{
					Error: err.Error(),
				}
				return
			}
			OBDInfoMessage.IPOS.Info = info
			respChan <- OBDInfoMessage
		}

		resp, err := adm.executeMethod(ctx, "GET", requestData{
			relPath:     adminAPIPrefix + "/obdinfo",
			queryValues: v,
		})

		defer closeResponse(resp)
		if err != nil {
			respChan <- OBDInfo{
				Error: err.Error(),
			}
			close(respChan)
			return
		}

		if resp.StatusCode != http.StatusOK {
			respChan <- OBDInfo{
				Error: httpRespToErrorResponse(resp).Error(),
			}
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			err := decoder.Decode(&OBDInfoMessage)
			OBDInfoMessage.TimeStamp = time.Now()

			if err == io.EOF {
				break
			}
			if err != nil {
				respChan <- OBDInfo{
					Error: err.Error(),
				}
			}
			respChan <- OBDInfoMessage
		}

		respChan <- OBDInfoMessage
		close(respChan)
	}()
	return respChan

}
