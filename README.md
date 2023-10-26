<img src="https://corethreat.net/ct_logo_big.png" height="150px"> 

**Triage - Live Forensic System Information Collector**

Triage is a Go program designed for performing live forensic analysis on a target system. It collects system information and stores it in a structured format for real-time analysis and forensic purposes. Triage retrieves various details about the system, such as running processes, network connections, auto-run configurations, and more, and then saves this information in a JSON log file. Additionally, it archives the collected data in a ZIP file for easy transport and storage.

![Screenshot](https://github.com/ipcis/CoreThreat_Triage_Go/blob/main/screen01.png)

**Key Features:**

- **Live Forensic Analysis:** Triage is designed for conducting live forensic investigations on a running system, making it a valuable tool for incident responders and digital forensics professionals.

- **Comprehensive System Information:** It collects an array of system information, including:
  - Date and time of data collection
  - System UUID (Universally Unique Identifier)
  - Operating System details
  - Computer name
  - Architecture (for Linux systems)
  - Running processes with their Process IDs (PIDs), names, and MD5 hashes
  - Network connections, including local and remote IPs and ports
  - Auto-run configurations with types, locations, image paths, and arguments (Windows only)
  - Prefetch files (Windows only)
  - Copied executables with their MD5 hashes

- **Structured Data Format:** Triage organizes data into a structured JSON format, making it easy to perform detailed analysis and investigations.

- **JSON Log File:** It creates a separate JSON log file with a unique name based on date and UUID for in-depth examination.

- **ZIP Archiving:** All collected data, including the JSON log file, is archived in a ZIP file, ensuring easy storage and sharing.

**Usage:**

1. Build and run the Triage program on the target system.
3. Triage will collect system information, create a JSON log file, and archive the data in a ZIP file.
4. The JSON log file contains detailed system information for live forensic analysis.
5. The ZIP archive contains all the data, including the JSON log file, for easy transport and storage.


**How to build:**
1. git clone https://github.com/ipcis/CoreThreat_Triage_Go.git
2. cd CoreThreat_Triage_Go
3. go mod init triage
4. go mod init tidy
5. go get github.com/botherder/go-autoruns
6. go get github.com/botherder/go-savetime/files
7. go get github.com/google/uuid
8. go get github.com/shirou/gopsutil/net
9. go get github.com/shirou/gopsutil/process
10. go build triage.go
11. chmod 777 triage


**Dependencies:**

- Go libraries (see import statements in the code)
- [shirou/gopsutil](https://github.com/shirou/gopsutil) for system information retrieval
- [botherder/go-autoruns](https://github.com/botherder/go-autoruns) for collecting auto-run configurations (Windows)
- [botherder/go-savetime](https://github.com/botherder/go-savetime) for file copying
- [google/uuid](https://github.com/google/uuid) for generating UUIDs

**Note:**

- Triage is designed for live forensic analysis and educational purposes. Always ensure that you comply with all legal and ethical considerations when using this tool.

**What comes next?**

- upload parameter to upload the zip data to a webserver
- capture the RAM

**License:**

Triage is provided under the [MIT License](LICENSE).

