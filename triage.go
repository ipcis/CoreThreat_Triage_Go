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

	"github.com/botherder/go-autoruns"
	"github.com/botherder/go-savetime/files"

	"archive/zip"
)

type SystemInfo struct {
	DateTime           string
	UUID               string
	OS                 string
	ComputerName       string
	Architecture       string
	RunningProcesses   []ProcessInfo
	NetworkConnections []NetworkConnectionInfo
	PrefetchFiles      []PrefetchInfo
	AutoRuns           []AutorunInfo
	CopiedExecutables  []CopiedExecutableInfo // New field for copied files with MD5 hashes
}

type CopiedExecutableInfo struct {
	FileName string
	MD5Hash  string
}

type AutorunInfo struct {
	Type      string
	Location  string
	ImagePath string
	Arguments string
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
	copyExecutables(info.RunningProcesses, info.AutoRuns, filepath.Join(info.UUID, "executables"), &info)

	// Serialize system information to JSON
	jsonData, err := json.Marshal(info)
	if err != nil {
		fmt.Println("Fehler beim Marshalling der Daten:", err)
		return
	}

	// Write JSON data to a log file
	logFileName := createLogFileName(info.DateTime, info.UUID)
	logFilePath := filepath.Join(info.UUID, logFileName)

	if logFilePath != "" {
		err := writeToFile(logFilePath, string(jsonData))
		if err != nil {
			fmt.Println("Fehler beim Schreiben in die Datei:", err)
		} else {
			fmt.Printf("JSON-Daten in die Datei %s geschrieben.\n", logFilePath)
		}
	}

	// Create a ZIP file of the directory
	zipFileName := info.UUID + ".zip"
	if err := createZipFile(info.UUID, zipFileName, info); err != nil {
		fmt.Println("Fehler beim Erstellen der ZIP-Datei:", err)
		return
	}

	fmt.Println("ZIP-Datei erstellt:", zipFileName)
	fmt.Println(string(jsonData))
}

func createZipFile(sourceDir, zipFileName string, info SystemInfo) error {
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a new zip file header
		zipHeader, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the name to be the relative path within the ZIP file
		zipHeader.Name, err = filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			zipHeader.Name += string(os.PathSeparator)
		}

		// Create the entry in the ZIP file
		zipEntry, err := zipWriter.CreateHeader(zipHeader)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Copy file contents to the ZIP entry
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(zipEntry, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func getSystemInfo() SystemInfo {
	var info SystemInfo

	info.DateTime = time.Now().Format("2006-01-02-15-04-05")
	info.UUID = generateUUID()
	info.OS = runtime.GOOS

	uuidStr := info.UUID

	parentDir := "./" + uuidStr
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		fmt.Println("Fehler beim Erstellen des übergeordneten Verzeichnisses:", err)
	}

	destDir := parentDir + "/executables"
	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Println("Fehler beim Erstellen des Zielverzeichnisses:", err)
	}

	fmt.Println("Verzeichnisse wurden erfolgreich erstellt.")

	if runtime.GOOS == "windows" {
		info.ComputerName = getWindowsComputerName()
		info.PrefetchFiles = getPrefetchFiles()
		info.AutoRuns = getAutoRuns()
	} else if runtime.GOOS == "linux" {
		info.ComputerName, info.Architecture = getLinuxSystemInfo()
		info.AutoRuns = getAutoRuns()
	}

	info.RunningProcesses = getRunningProcesses()
	info.NetworkConnections = getNetworkConnections()

	return info
}

func copyExecutables(processes []ProcessInfo, autoruns []AutorunInfo, destDir string, info *SystemInfo) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Println("Fehler beim Erstellen des Zielverzeichnisses:", err)
		return
	}

	for _, process := range processes {
		if process.ExeMD5 != "N/A" {
			srcPath := process.Name
			dstPath := filepath.Join(destDir, filepath.Base(srcPath)+".bin")

			if err := files.Copy(srcPath, dstPath); err != nil {
				fmt.Printf("Fehler beim Kopieren von %s: %s\n", srcPath, err)
			} else {
				md5Hash := calculateMD5(dstPath)
				info.CopiedExecutables = append(info.CopiedExecutables, CopiedExecutableInfo{
					FileName: filepath.Base(srcPath) + ".bin",
					MD5Hash:  md5Hash,
				})
			}
		}
	}

	for _, autorun := range autoruns {
		if autorun.ImagePath != "" {
			srcPath := autorun.ImagePath
			dstPath := filepath.Join(destDir, filepath.Base(srcPath)+".bin")

			if err := files.Copy(srcPath, dstPath); err != nil {
				fmt.Printf("Fehler beim Kopieren von %s: %s\n", srcPath, err)
			} else {
				md5Hash := calculateMD5(dstPath)
				info.CopiedExecutables = append(info.CopiedExecutables, CopiedExecutableInfo{
					FileName: filepath.Base(srcPath) + ".bin",
					MD5Hash:  md5Hash,
				})
			}
		}
	}
}

func getAutoRuns() []AutorunInfo {
	autoruns := autoruns.Autoruns()
	var autorunInfo []AutorunInfo

	for _, autorun := range autoruns {
		autorunInfo = append(autorunInfo, AutorunInfo{
			Type:      autorun.Type,
			Location:  autorun.Location,
			ImagePath: autorun.ImagePath,
			Arguments: autorun.Arguments,
		})
	}

	return autorunInfo
}

func getPrefetchFiles() []PrefetchInfo {
	prefetchDir := `C:\Windows\Prefetch`
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
