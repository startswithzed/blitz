package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type CPUInfo struct {
	Model      string  `json:"model"`
	Cores      int     `json:"cores"`
	Arch       string  `json:"arch"`
	ClockSpeed float64 `json:"clockSpeed"`
}

type MemInfo struct {
	Total uint64 `json:"total"`
}

type OSInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

type SystemInfo struct {
	CPU CPUInfo `json:"cpu"`
	Mem MemInfo `json:"memory"`
	OS  OSInfo  `json:"os"`
}

type Metrics struct {
	CPU float64 `json:"cpu"`
	Mem float64 `json:"mem"`
}

func getSystemInfo() SystemInfo {

	// cpu info
	cpuCores := runtime.NumCPU()
	cpuArch := runtime.GOARCH
	cpuInfo, _ := cpu.Info() // TODO: Handle error
	cpuModel := cpuInfo[0].ModelName
	cpuClockSpeed := cpuInfo[0].Mhz
	cpuI := CPUInfo{
		Model:      cpuModel,
		Cores:      cpuCores,
		Arch:       cpuArch,
		ClockSpeed: cpuClockSpeed,
	}

	// mem info
	memInfo, _ := mem.VirtualMemory() // TODO: Handle error
	memI := MemInfo{Total: memInfo.Total}

	// os info
	hostInfo, _ := host.Info() // TODO: Handle error
	osName := hostInfo.PlatformFamily
	osArch := hostInfo.Platform
	osVersion := hostInfo.PlatformVersion
	osI := OSInfo{
		Name:    osName,
		Version: osVersion,
		Arch:    osArch,
	}

	return SystemInfo{
		CPU: cpuI,
		Mem: memI,
		OS:  osI,
	}
}

func getSysInfoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("INFO: got GET system info request\n")

	sysInfo := getSystemInfo()

	bytes, err := json.Marshal(sysInfo)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bytes)
	if err != nil {
		fmt.Println("ERROR:  could not write response")
	}
}

func getMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")

	ticker := time.NewTicker(500 * time.Millisecond) // TODO: let user specify this duration

	for {
		select {
		case <-ticker.C:
			percentage, err := cpu.Percent(time.Second, false)
			if err != nil {
				fmt.Println(err)
			}

			memInfo, err := mem.VirtualMemory()
			if err != nil {
				fmt.Println(err)
			}

			metrics := Metrics{
				CPU: percentage[0],
				Mem: memInfo.UsedPercent,
			}

			bytes, err := json.Marshal(metrics)
			if err != nil {
				fmt.Println(err)
			}

			eventData := append(bytes, '\n')
			_, err = w.Write(eventData)
			if err != nil {
				fmt.Println("Error writing SSE event:", err)
				return
			}

			// flush the response writer to ensure data is sent immediately
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			ticker.Stop()
			return
		}
	}
}
