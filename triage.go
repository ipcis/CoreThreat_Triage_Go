package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

type SystemInfo struct {
	DateTime           string
	UUID               string
	OS                 string
	ComputerName       string
	Architecture       string
	RunningProcesses   []ProcessInfo
	NetworkConnections []NetworkConnectionInfo
	PrefetchFiles      []PrefetchInfo // Neues Feld für Prefetch-Dateien
}

type ProcessInfo struct {
	PID    int32
	Name   string
	ExeMD5 string
}

type NetworkConnectionInfo struct {
	FD         uint32
	Family     uint32
	Type       uint32
	LocalIP    string
	LocalPort  uint32
	RemoteIP   string
	RemotePort uint32
}

type PrefetchInfo struct {
	FileName  string
	Timestamp string
}

var md5Cache sync.Map

func main() {
	info := getSystemInfo()
	jsonData, err := json.Marshal(info)
	if err != nil {
		fmt.Println("Fehler beim Marshalling der Daten:", err)
		return
	}

	logFileName := createLogFileName(info.DateTime, info.ComputerName)
	logFilePath := filepath.Join(".", logFileName)
	if logFilePath != "" {
		err := writeToFile(logFilePath, string(jsonData))
		if err != nil {
			fmt.Println("Fehler beim Schreiben in die Datei:", err)
		} else {
			fmt.Printf("JSON-Daten in die Datei %s geschrieben.\n", logFilePath)
		}
	}

	fmt.Println(string(jsonData))
}

func getSystemInfo() SystemInfo {
	var info SystemInfo

	info.DateTime = time.Now().Format("2006-01-02-15-04-05")
	info.UUID = generateUUID()
	info.OS = runtime.GOOS

	if runtime.GOOS == "windows" {
		info.ComputerName = getWindowsComputerName()
		info.PrefetchFiles = getPrefetchFiles() // Füge Prefetch-Dateien hinzu
	} else if runtime.GOOS == "linux" {
		info.ComputerName, info.Architecture = getLinuxSystemInfo()
	}

	info.RunningProcesses = getRunningProcesses()
	info.NetworkConnections = getNetworkConnections()

	return info
}

func getPrefetchFiles() []PrefetchInfo {
	prefetchDir := `C:\Windows\Prefetch` // Pfade variieren je nach Windows-Version
	var prefetchInfo []PrefetchInfo

	err := filepath.Walk(prefetchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			prefetchInfo = append(prefetchInfo, PrefetchInfo{
				FileName:  info.Name(),
				Timestamp: info.ModTime().String(),
			})
		}
		return nil
	})

	if err != nil {
		fmt.Println("Fehler beim Abrufen der Prefetch-Dateien:", err)
	}

	return prefetchInfo
}

func generateUUID() string {
	u := uuid.New()
	return u.String()
}

func getWindowsComputerName() string {
	cmd := exec.Command("hostname")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "N/A"
}

func getLinuxSystemInfo() (string, string) {
	cmd := exec.Command("uname", "-n", "-m")
	out, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) == 2 {
			return fields[0], fields[1]
		}
	}
	return "N/A", "N/A"
}

func getRunningProcesses() []ProcessInfo {
	var processes []ProcessInfo

	pids, _ := process.Pids()
	for _, pid := range pids {
		proc, _ := process.NewProcess(pid)
		name, _ := proc.Name()
		exePath, _ := proc.Exe()

		md5sum, ok := md5Cache.Load(exePath)
		if !ok {
			md5sum = calculateMD5(exePath)
			md5Cache.Store(exePath, md5sum)
		}

		processes = append(processes, ProcessInfo{
			PID:    pid,
			Name:   name,
			ExeMD5: md5sum.(string),
		})
	}

	return processes
}

func calculateMD5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return "N/A"
	}
	defer file.Close()

	hasher := md5.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "N/A"
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func createLogFileName(dateTime, computerName string) string {
	return fmt.Sprintf("system_info_%s_%s.log", dateTime, computerName)
}

func writeToFile(filePath, data string) error {
	dataWithDateTime := data + "\n"
	return ioutil.WriteFile(filePath, []byte(dataWithDateTime), 0644)
}

func getNetworkConnections() []NetworkConnectionInfo {
	var connections []NetworkConnectionInfo

	stats, _ := net.Connections("inet")
	for _, stat := range stats {
		connections = append(connections, NetworkConnectionInfo{
			FD:         stat.Fd,
			Family:     stat.Family,
			Type:       stat.Type,
			LocalIP:    stat.Laddr.IP,
			LocalPort:  stat.Laddr.Port,
			RemoteIP:   stat.Raddr.IP,
			RemotePort: stat.Raddr.Port,
		})
	}

	return connections
}
