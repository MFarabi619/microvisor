package healthcheck

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"strings"

	"github.com/joho/godotenv"
	"github.com/xsachax/lazyos/pkg/config"
	"golang.org/x/crypto/ssh"
)

// Global connection cache: machine ID -> *ssh.Client
var connCache = struct {
	sync.Mutex
	m map[int]*ssh.Client
}{m: make(map[int]*ssh.Client)}

var debugLogFile *os.File

func init() {
	var err error 
	debugLogFile, err = os.OpenFile("lazyos.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		debugLogFile = os.Stderr
	}
}

type MachineInfo struct {
	ID       int
	Username string
	Hostname string
	Port     int
	Password string
	Status   string
	TermEnv  string // e.g., "xterm-kitty"
	AddKeysToAgent bool
	Uptime   string // stores output of 'uptime'
	OS       string // stores output of 'uname -s' or similar
	Kernel   string // stores output of 'uname -r'
	CPU      string // stores output of CPU info command
	Memory   string // stores output of memory info command
	Chassis  string // stores output of chassis info command
}

	// StartBackgroundStatusRetry launches a goroutine to keep retrying SSH connection for inactive/offline machines
	func StartBackgroundStatusRetry(m *MachineInfo, interval time.Duration, stopCh <-chan struct{}) {
	       go func() {
		       for {
			       select {
			       case <-stopCh:
				       return
			       default:
				       if m.Status == "Online" {
					       return
				       }
				       FetchStatus(m)
				       if m.Status == "Online" {
					       fmt.Fprintf(debugLogFile, "[INFO] Machine %d: Status changed to Online during background retry\n", m.ID)
					       return
				       }
				       time.Sleep(interval)
			       }
		       }
	       }()
	}

// LoadPasswordsFromEnv loads the password for a given machine ID from the environment variables
func LoadPasswordsFromEnv(machineID int) (string, error) {
	_ = godotenv.Load()
	key := fmt.Sprintf("PASSWORD_%d", machineID)
	pw := os.Getenv(key)
	if pw == "" {
		return "", fmt.Errorf("password for machine %d not found in .env", machineID)
	}
	return pw, nil
}

// FetchStatus tries to SSH and ping the machine, updating status accordingly
func ConvertMachines(cfgMachines []config.Machine) []MachineInfo {
	infos := make([]MachineInfo, len(cfgMachines))
	for i, m := range cfgMachines {
		infos[i] = MachineInfo{
			ID:            m.ID,
			Username:      m.Username,
			Hostname:      m.Hostname,
			Port:          m.Port,
            TermEnv:       "",
            AddKeysToAgent: m.AddKeysToAgent,
		}
	}
	return infos
}
// Helper functions for each fetch
func fetchUptime(client *ssh.Client, m *MachineInfo) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "", err
       }
       defer session.Close()
       out, err := session.Output("uptime")
       return string(out), err
}

func fetchOS(client *ssh.Client) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "unknown", err
       }
       defer session.Close()
       out, err := session.Output("uname -s")
       if err != nil {
	       return "unknown", err
       }
       return strings.TrimSpace(string(out)), nil
}

func fetchCPU(client *ssh.Client) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "unknown", err
       }
       defer session.Close()
       session.Setenv("TERM", "xterm-kitty")
       out, err := session.Output("lscpu | grep 'Model name' || sysctl -n machdep.cpu.brand_string || cat /proc/cpuinfo | grep 'model name' | head -1")
       if err != nil || len(out) == 0 {
	       return "unknown", err
       }
       cpuStr := strings.TrimSpace(string(out))
       cpuStr = strings.TrimPrefix(cpuStr, "Model name:")
       cpuStr = strings.TrimPrefix(cpuStr, "model name:")
       cpuStr = strings.TrimPrefix(cpuStr, "machdep.cpu.brand_string:")
       return strings.TrimSpace(cpuStr), nil
}

func fetchMemory(client *ssh.Client, osName string) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "unknown", err
       }
       defer session.Close()
       session.Setenv("TERM", "xterm-kitty")
       var out []byte
       var memStr string
       if strings.HasPrefix(strings.ToLower(osName), "darwin") || strings.HasPrefix(strings.ToLower(osName), "mac") {
	       out, err = session.Output("system_profiler SPHardwareDataType | /usr/bin/awk \"/Memory:/ {print $2 $3}\"")
	       rawProfiler := string(out)
	       memLines := strings.Split(rawProfiler, "\n")
	       for _, line := range memLines {
		       line = strings.TrimSpace(line)
		       if len(line) > 0 {
			       memStr = strings.ReplaceAll(line, "Memory:", "")
			       memStr = strings.ReplaceAll(memStr, " ", "")
			       break
		       }
	       }
	       if err == nil && len(memStr) > 0 {
		       return memStr, nil
	       }
	       return "unknown", err
       } else {
	       out, err = session.Output("free -g | awk '/Mem:/ {print $2 \"GB\"}' || cat /proc/meminfo | grep MemTotal || sysctl -n hw.memsize")
	       memStr = strings.TrimSpace(string(out))
	       if err == nil && len(memStr) > 0 {
		       if strings.Contains(memStr, "MemTotal") {
			       parts := strings.Fields(memStr)
			       if len(parts) >= 2 {
				       kb, parseErr := parseInt(parts[1])
				       if parseErr == nil {
					       gb := kb / 1024 / 1024
					       return fmt.Sprintf("%dGB", gb), nil
				       }
			       }
			       return "unknown", nil
		       } else if strings.Contains(memStr, "GB") {
			       return memStr, nil
		       } else {
			       bytes, parseErr := parseInt(memStr)
			       if parseErr == nil {
				       gb := bytes / 1024 / 1024 / 1024
				       return fmt.Sprintf("%dGB", gb), nil
			       }
			       return "unknown", nil
		       }
	       }
	       return "unknown", err
       }
}

func fetchKernel(client *ssh.Client) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "unknown", err
       }
       defer session.Close()
       out, err := session.Output("uname -r")
       if err != nil {
	       return "unknown", err
       }
       return strings.TrimSpace(string(out)), nil
}

func fetchChassis(client *ssh.Client, osName string) (string, error) {
       session, err := client.NewSession()
       if err != nil {
	       return "unknown", err
       }
       defer session.Close()
       out, err := session.Output("hostnamectl | grep Chassis")
       if err == nil && len(out) > 0 {
	       return strings.TrimSpace(string(out)), nil
       }
       chassisTypeOut, err2 := session.Output("cat /sys/class/dmi/id/chassis_type")
       if err2 == nil && len(chassisTypeOut) > 0 {
	       dmidecodeOut, errDmi := session.Output("dmidecode -s system-product-name")
	       if errDmi == nil && len(dmidecodeOut) > 0 {
		       return strings.TrimSpace(string(dmidecodeOut)), nil
	       }
	       productNameOut, errProd := session.Output("cat /sys/class/dmi/id/product_name")
	       if errProd == nil && len(productNameOut) > 0 {
		       return strings.TrimSpace(string(productNameOut)), nil
	       }
	       chassisType := strings.TrimSpace(string(chassisTypeOut))
	       switch chassisType {
	       case "8":
		       return "laptop", nil
	       case "3":
		       return "desktop", nil
	       case "10":
		       return "tablet", nil
	       case "9":
		       return "notebook", nil
	       default:
		       return "unknown", nil
	       }
       }
       // macOS hardware model
       modelSession, sessErr := client.NewSession()
       if sessErr == nil {
	       defer modelSession.Close()
	       modelOut, err3 := modelSession.Output("sysctl -n hw.model")
	       if err3 == nil && len(modelOut) > 0 {
		       return strings.TrimSpace(string(modelOut)), nil
	       }
       }
       profilerSession, sessErr2 := client.NewSession()
       if sessErr2 == nil {
	       defer profilerSession.Close()
	       profilerOut, err4 := profilerSession.Output("system_profiler SPHardwareDataType | grep 'Model Identifier'")
	       if err4 == nil && len(profilerOut) > 0 {
		       return strings.TrimSpace(string(profilerOut)), nil
	       }
       }
       return "unknown", nil
}

func FetchStatus(m *MachineInfo) {
       pw, err := LoadPasswordsFromEnv(m.ID)
       maskedPw := "(empty)"
       if len(pw) > 0 {
	       if len(pw) > 4 {
		       maskedPw = pw[:2] + "***" + pw[len(pw)-2:]
	       } else {
		       maskedPw = "***"
	       }
       }
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Loaded password='%s', err=%v\n", m.ID, maskedPw, err)
       if err != nil {
	       if m.Status == "" {
		       m.Status = "Inactive"
	       } else {
		       m.Status = "Offline"
	       }
	       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Password not found, status=%s\n", m.ID, m.Status)
	       return
       }
       m.Password = pw
       config := &ssh.ClientConfig{
	       User: m.Username,
	       Auth: []ssh.AuthMethod{
		       ssh.Password(m.Password),
	       },
	       HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	       Timeout: 1 * time.Second,
       }
       addr := fmt.Sprintf("%s:%d", m.Hostname, m.Port)

       var client *ssh.Client
       maxAttempts := 3
       backoff := 500 * time.Millisecond
       for attempt := 1; attempt <= maxAttempts; attempt++ {
	       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Attempting SSH dial to %s (attempt %d/%d)\n", m.ID, addr, attempt, maxAttempts)
	       client, err = ssh.Dial("tcp", addr, config)
	       if err == nil {
		       break
	       }
	       if attempt < maxAttempts {
		       time.Sleep(backoff)
		       backoff *= 2
	       }
       }
       if err != nil {
	       if m.Status == "" {
		       m.Status = "Inactive"
	       } else {
		       m.Status = "Offline"
	       }
	       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: SSH dial failed after %d attempts, status=%s, err=%v\n", m.ID, maxAttempts, m.Status, err)
	       return
       }
       connCache.Lock()
       connCache.m[m.ID] = client
       connCache.Unlock()

       // Fetch Uptime
       uptimeOut, err := fetchUptime(client, m)
       if err != nil {
	       if m.Status == "Inactive" {
		       m.Status = "Offline"
	       } else {
		       m.Status = "Inactive"
	       }
	       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime failed, status=%s, err=%v\n", m.ID, m.Status, err)
	       connCache.Lock()
	       if cached, ok := connCache.m[m.ID]; ok && cached == client {
		       cached.Close()
		       delete(connCache.m, m.ID)
	       }
	       connCache.Unlock()
	       return
       }
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime fetched: %s\n", m.ID, uptimeOut)
       m.Status = "Online"
       m.Uptime = ParseUptime(uptimeOut)

       // Fetch OS
       m.OS, _ = fetchOS(client)
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: OS info fetched: %s\n", m.ID, m.OS)

       // Fetch CPU
       m.CPU, _ = fetchCPU(client)
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: CPU info fetched: %s\n", m.ID, m.CPU)

       // Fetch Memory
       m.Memory, _ = fetchMemory(client, m.OS)
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Memory info fetched: %s\n", m.ID, m.Memory)

       // Fetch Kernel
       m.Kernel, _ = fetchKernel(client)
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Kernel info fetched: %s\n", m.ID, m.Kernel)

       // Fetch Chassis
       m.Chassis, _ = fetchChassis(client, m.OS)
       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Chassis info fetched: %s\n", m.ID, m.Chassis)

       fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime succeeded, status=%s, uptime=%s, os=%s, kernel=%s, chassis=%s\n", m.ID, m.Status, m.Uptime, m.OS, m.Kernel, m.Chassis)
}

// parseInt parses a string to int64, returns error if not possible
func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}