package healthcheck

import (
	"fmt"
	"os"
	"sync"
	"time"

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
		// Store new connection in cache
		connCache.Lock()
		connCache.m[m.ID] = client
		connCache.Unlock()
	}

	// health check: run 'uptime' on remote machine
	session, err := client.NewSession()
	if err != nil {
		m.Status = "Offline"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: NewSession failed, status=%s, err=%v\n", m.ID, m.Status, err)
		// Remove broken connection from cache
		connCache.Lock()
		if cached, ok := connCache.m[m.ID]; ok && cached == client {
			cached.Close()
			delete(connCache.m, m.ID)
		}
		connCache.Unlock()
		return
	}
	defer session.Close()

	if err := session.Run("uptime"); err != nil {
		if m.Status == "Inactive" {
			m.Status = "Offline"
		} else {
			m.Status = "Inactive"
		}
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime failed, status=%s, err=%v\n", m.ID, m.Status, err)
		// Remove broken connection from cache
		connCache.Lock()
		if cached, ok := connCache.m[m.ID]; ok && cached == client {
			cached.Close()
			delete(connCache.m, m.ID)
		}
		connCache.Unlock()
		return
	}
	m.Status = "Online"
	fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime succeeded, status=%s\n", m.ID, m.Status)
}
