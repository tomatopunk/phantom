// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package server

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
)

func collectHostMetrics(context.Context) *proto.GetHostMetricsResponse {
	resp := &proto.GetHostMetricsResponse{}
	var errs []string

	if h, err := os.Hostname(); err == nil {
		resp.Hostname = h
	} else if b, err := os.ReadFile("/proc/sys/kernel/hostname"); err == nil {
		resp.Hostname = strings.TrimSpace(string(b))
	}

	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(b))
		if len(fields) >= 3 {
			if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
				resp.LoadavgOne = v
			}
			if v, err := strconv.ParseFloat(fields[1], 64); err == nil {
				resp.LoadavgFive = v
			}
			if v, err := strconv.ParseFloat(fields[2], 64); err == nil {
				resp.LoadavgFifteen = v
			}
		}
	} else {
		errs = append(errs, "loadavg: "+err.Error())
	}

	parseMeminfo(resp, &errs)
	parseProcStatCPUs(resp, &errs)
	parseNetDev(resp, &errs)

	if len(errs) > 0 {
		resp.ErrorMessage = strings.Join(errs, "; ")
	}
	return resp
}

func parseMeminfo(resp *proto.GetHostMetricsResponse, errs *[]string) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		*errs = append(*errs, "meminfo: "+err.Error())
		return
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		rest := strings.TrimSpace(line[i+1:])
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		kb, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "MemTotal":
			resp.MemTotalKb = kb
		case "MemAvailable":
			resp.MemAvailableKb = kb
		case "Buffers":
			resp.MemBuffersKb = kb
		case "Cached":
			resp.MemCachedKb = kb
		case "SwapTotal":
			resp.MemSwapTotalKb = kb
		case "SwapFree":
			resp.MemSwapFreeKb = kb
		}
	}
}

func parseProcStatCPUs(resp *proto.GetHostMetricsResponse, errs *[]string) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		*errs = append(*errs, "stat: "+err.Error())
		return
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			if len(resp.Cpus) > 0 {
				break
			}
			continue
		}
		label := fields[0]
		if !isProcStatCPULine(label) {
			if len(resp.Cpus) > 0 {
				break
			}
			continue
		}
		cj := &proto.CpuJiffies{Label: label}
		parseJiffies(fields[1:], cj)
		resp.Cpus = append(resp.Cpus, cj)
	}
}

func isProcStatCPULine(label string) bool {
	if label == "cpu" {
		return true
	}
	if len(label) < 4 || !strings.HasPrefix(label, "cpu") {
		return false
	}
	return label[3] >= '0' && label[3] <= '9'
}

func parseJiffies(nums []string, cj *proto.CpuJiffies) {
	u64 := func(i int) uint64 {
		if i >= len(nums) {
			return 0
		}
		v, _ := strconv.ParseUint(nums[i], 10, 64)
		return v
	}
	cj.User = u64(0)
	cj.Nice = u64(1)
	cj.System = u64(2)
	cj.Idle = u64(3)
	cj.Iowait = u64(4)
	cj.Irq = u64(5)
	cj.Softirq = u64(6)
	cj.Steal = u64(7)
	cj.Guest = u64(8)
	cj.GuestNice = u64(9)
}

func parseNetDev(resp *proto.GetHostMetricsResponse, errs *[]string) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		*errs = append(*errs, "net/dev: "+err.Error())
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		if lineNo <= 2 {
			continue
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		ifName := strings.TrimSpace(line[:colon])
		rest := strings.TrimSpace(line[colon+1:])
		fields := strings.Fields(rest)
		if len(fields) < 16 {
			continue
		}
		parseU64 := func(s string) uint64 {
			v, _ := strconv.ParseUint(s, 10, 64)
			return v
		}
		resp.NetDevs = append(resp.NetDevs, &proto.NetDevRow{
			Name:        ifName,
			RxBytes:     parseU64(fields[0]),
			RxPackets:   parseU64(fields[1]),
			RxErrors:    parseU64(fields[2]),
			RxDropped:   parseU64(fields[3]),
			TxBytes:     parseU64(fields[8]),
			TxPackets:   parseU64(fields[9]),
			TxErrors:    parseU64(fields[10]),
			TxDropped:   parseU64(fields[11]),
		})
	}
	if err := sc.Err(); err != nil {
		*errs = append(*errs, "net/dev scan: "+err.Error())
	}
}

func collectTaskTree(_ context.Context, tgid uint32) *proto.GetTaskTreeResponse {
	resp := &proto.GetTaskTreeResponse{Tgid: tgid}
	if tgid == 0 {
		resp.ErrorMessage = "tgid must be non-zero"
		return resp
	}
	taskRoot := filepath.Join("/proc", strconv.FormatUint(uint64(tgid), 10), "task")
	ents, err := os.ReadDir(taskRoot)
	if err != nil {
		resp.ErrorMessage = fmt.Sprintf("read %s: %v", taskRoot, err)
		return resp
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		tid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		tid := uint32(tid64)
		info := parseTaskStatus(filepath.Join(taskRoot, e.Name(), "status"))
		if info == nil {
			info = &proto.TaskInfo{Tid: tid}
		} else {
			info.Tid = tid
		}
		resp.Tasks = append(resp.Tasks, info)
	}
	return resp
}

func parseTaskStatus(path string) *proto.TaskInfo {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	info := &proto.TaskInfo{}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "Name":
			info.Name = val
		case "State":
			info.State = val
		case "VmPeak":
			if kb := parseKbSuffix(val); kb >= 0 {
				info.VmPeakKb = kb
			}
		case "VmSize":
			if kb := parseKbSuffix(val); kb >= 0 {
				info.VmSizeKb = kb
			}
		case "VmRSS":
			if kb := parseKbSuffix(val); kb >= 0 {
				info.VmRssKb = kb
			}
		case "VmHWM":
			if kb := parseKbSuffix(val); kb >= 0 {
				info.VmHwmKb = kb
			}
		case "Threads":
			if n, err := strconv.ParseInt(strings.Fields(val)[0], 10, 32); err == nil {
				info.ThreadsCount = int32(n)
			}
		}
	}
	return info
}

func parseKbSuffix(s string) int64 {
	fields := strings.Fields(s)
	if len(fields) < 1 {
		return -1
	}
	n, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return -1
	}
	return n
}
