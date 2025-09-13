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

// FetchLocationWithRetry tries to fetch the machine's location via IP geolocation API, with retries, and approximates to nearest city
func FetchLocationWithRetry(hostname string, maxRetries int) (lat float64, lon float64, city string, err error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		lat, lon, city, lastErr = fetchLocationFromAPI(hostname)
		if lastErr == nil {
			return lat, lon, city, nil
		}
		time.Sleep(time.Duration(1+attempt) * time.Second)
	}
	return 0, 0, "unknown", lastErr
}

// fetchLocationFromAPI is a stub for actual geolocation API call
func fetchLocationFromAPI(hostname string) (float64, float64, string, error) {
	// TODO: Implement actual API call
	return 43.6532, -79.3832, "Toronto", nil
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
		Timeout: 3 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", m.Hostname, m.Port)

	// Try to reuse connection from cache
	var client *ssh.Client
	connCache.Lock()
	cached, ok := connCache.m[m.ID]
	connCache.Unlock()
	if ok && cached != nil {
		session, err := cached.NewSession()
		if err == nil {
			defer session.Close()
			if err := session.Run("true"); err == nil {
				client = cached
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Reusing cached SSH connection\n", m.ID)

		// Fetch CPU info (try lscpu, then sysctl, then /proc/cpuinfo)
		cpuSession, err := client.NewSession()
		if err == nil {
			defer cpuSession.Close()
			cpuSession.Setenv("TERM", "xterm-kitty")
			var cpuOut []byte
			cpuOut, err = cpuSession.Output("lscpu | grep 'Model name' || sysctl -n machdep.cpu.brand_string || cat /proc/cpuinfo | grep 'model name' | head -1")
			if err == nil && len(cpuOut) > 0 {
				cpuStr := strings.TrimSpace(string(cpuOut))
				cpuStr = strings.TrimPrefix(cpuStr, "Model name:")
				cpuStr = strings.TrimPrefix(cpuStr, "model name:")
				cpuStr = strings.TrimPrefix(cpuStr, "machdep.cpu.brand_string:")
				m.CPU = strings.TrimSpace(cpuStr)
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: CPU info fetched: %s\n", m.ID, m.CPU)
			} else {
				m.CPU = "unknown"
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: CPU info fetch failed, err=%v, output=%s\n", m.ID, err, string(cpuOut))
			}
		} else {
			m.CPU = "unknown"
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: CPU session failed, err=%v\n", m.ID, err)
		}

		// Fetch Memory info (try free -g, sysctl, /proc/meminfo)
		memSession, err := client.NewSession()
		if err == nil {
			defer memSession.Close()
			memSession.Setenv("TERM", "xterm-kitty")
			var memOut []byte
			memOut, err = memSession.Output("free -g | awk '/Mem:/ {print $2 \"GB\"}' || sysctl -n hw.memsize || cat /proc/meminfo | grep MemTotal")
			memStr := strings.TrimSpace(string(memOut))
			if err == nil && len(memStr) > 0 {
				if strings.Contains(memStr, "MemTotal") {
					// /proc/meminfo output: MemTotal:   16333736 kB
					parts := strings.Fields(memStr)
					if len(parts) >= 2 {
						kb, parseErr := parseInt(parts[1])
						if parseErr == nil {
							gb := kb / 1024 / 1024
							m.Memory = fmt.Sprintf("%dGB", gb)
						} else {
							m.Memory = "unknown"
						}
					} else {
						m.Memory = "unknown"
					}
				} else if strings.Contains(memStr, "GB") {
					m.Memory = memStr
				} else if strings.HasSuffix(m.OS, "Darwin") || strings.HasSuffix(m.OS, "Mac") {
					// Darwin/macOS: sysctl -n hw.memsize returns bytes
					bytes, parseErr := parseInt(memStr)
					if parseErr == nil {
						gb := bytes / 1024 / 1024 / 1024
						m.Memory = fmt.Sprintf("%dGB", gb)
					} else {
						m.Memory = "unknown"
					}
				} else {
					// Try to parse as bytes for other OSes
					bytes, parseErr := parseInt(memStr)
					if parseErr == nil {
						gb := bytes / 1024 / 1024 / 1024
						m.Memory = fmt.Sprintf("%dGB", gb)
					} else {
						m.Memory = "unknown"
					}
				}
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Memory info fetched: %s\n", m.ID, m.Memory)
			} else {
				m.Memory = "unknown"
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Memory info fetch failed, err=%v, output=%s\n", m.ID, err, memStr)
			}
		} else {
			m.Memory = "unknown"
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Memory session failed, err=%v\n", m.ID, err)
		}

			} else {
				cached.Close()
				connCache.Lock()
				delete(connCache.m, m.ID)
				connCache.Unlock()
			}
		} else {
			cached.Close()
			connCache.Lock()
			delete(connCache.m, m.ID)
			connCache.Unlock()
		}
	}
	if client == nil {
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Attempting new SSH dial to %s\n", m.ID, addr)
		client, err = ssh.Dial("tcp", addr, config)
		if err != nil {
			if m.Status == "" {
				m.Status = "Inactive"
			} else {
				m.Status = "Offline"
			}
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: SSH dial failed, status=%s, err=%v\n", m.ID, m.Status, err)
			return
		}
		connCache.Lock()
		connCache.m[m.ID] = client
		connCache.Unlock()
	}

	session, err := client.NewSession()
	if err != nil {
		m.Status = "Offline"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: NewSession failed, status=%s, err=%v\n", m.ID, m.Status, err)
		connCache.Lock()
		if cached, ok := connCache.m[m.ID]; ok && cached == client {
			cached.Close()
			delete(connCache.m, m.ID)
		}
		connCache.Unlock()
		return
	}
	defer session.Close()

	var uptimeOut []byte
	uptimeOut, err = session.Output("uptime")
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
	fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime fetched: %s\n", m.ID, string(uptimeOut))
	m.Status = "Online"
	m.Uptime = ParseUptime(string(uptimeOut))

	// Fetch OS info
	osSession, err := client.NewSession()
	if err == nil {
		defer osSession.Close()
		osOut, err := osSession.Output("uname -s")
		if err == nil {
			m.OS = strings.TrimSpace(string(osOut))
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: OS info fetched: %s\n", m.ID, m.OS)
		} else {
			m.OS = "unknown"
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: OS info fetch failed, err=%v, output=%s\n", m.ID, err, string(osOut))
		}
	} else {
		m.OS = "unknown"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: OS session failed, err=%v\n", m.ID, err)
	}

	// Fetch Kernel info
	kernelSession, err := client.NewSession()
	if err == nil {
		defer kernelSession.Close()
		kernelOut, err := kernelSession.Output("uname -r")
		if err == nil {
			m.Kernel = strings.TrimSpace(string(kernelOut))
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Kernel info fetched: %s\n", m.ID, m.Kernel)
		} else {
			m.Kernel = "unknown"
			fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Kernel info fetch failed, err=%v, output=%s\n", m.ID, err, string(kernelOut))
		}
	} else {
		m.Kernel = "unknown"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Kernel session failed, err=%v\n", m.ID, err)
	}

	// Fetch Chassis info
	chassisSession, err := client.NewSession()
	if err == nil {
		defer chassisSession.Close()
		var chassis string
		chassisOut, err := chassisSession.Output("hostnamectl | grep Chassis")
		if err == nil && len(chassisOut) > 0 {
			chassis = strings.TrimSpace(string(chassisOut))
		} else {
			chassisTypeOut, err2 := chassisSession.Output("cat /sys/class/dmi/id/chassis_type")
			if err2 == nil && len(chassisTypeOut) > 0 {
				chassisType := strings.TrimSpace(string(chassisTypeOut))
				switch chassisType {
				case "8":
					chassis = "laptop"
				case "3":
					chassis = "desktop"
				case "10":
					chassis = "tablet"
				case "9":
					chassis = "notebook"
				default:
					chassis = "unknown"
				}
			} else {
				chassis = "unknown"
			}
		}
		m.Chassis = chassis
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Chassis info fetched: %s\n", m.ID, m.Chassis)
	} else {
		m.Chassis = "unknown"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Chassis session failed, err=%v\n", m.ID, err)
	}

	fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime succeeded, status=%s, uptime=%s, os=%s, kernel=%s, chassis=%s\n", m.ID, m.Status, m.Uptime, m.OS, m.Kernel, m.Chassis)
}

// parseInt parses a string to int64, returns error if not possible
func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}