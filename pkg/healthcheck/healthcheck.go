package healthcheck

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xsachax/lazyos/pkg/config"
	"golang.org/x/crypto/ssh"
)

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
    SetEnvVars map[string]string // for extensibility, e.g. {"TERM": "xterm-kitty"}
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
// Machines is a slice of MachineInfo representing the SSH config entries
// ConvertMachines converts []main.Machine to []MachineInfo
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
			SetEnvVars:    m.SetEnv,
		}
		// If TERM is set in env, set TermEnv for convenience
		if term, ok := m.SetEnv["TERM"]; ok {
			infos[i].TermEnv = term
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
	fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Attempting SSH to %s\n", m.ID, addr)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		if m.Status == "" {
			m.Status = "Inactive"
		} else {
			m.Status = "Offline"
		}
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: SSH dial failed, status=%s, err=%v\n", m.ID, m.Status, err)
		return
	}
	defer client.Close()

	// health check: run 'uptime' on remote machine
	session, err := client.NewSession()
	if err != nil {
		m.Status = "Offline"
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: NewSession failed, status=%s, err=%v\n", m.ID, m.Status, err)
		return
	}
	defer session.Close()

	if m.SetEnvVars != nil {
		for k, v := range m.SetEnvVars {
			if err := session.Setenv(k, v); err != nil {
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Failed to set env %s=%s, err=%v\n", m.ID, k, v, err)
			} else {
				fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: Set env %s=%s\n", m.ID, k, v)
			}
		}
	}

	if err := session.Run("uptime"); err != nil {
		if m.Status == "Inactive" {
			m.Status = "Offline"
		} else {
			m.Status = "Inactive"
		}
		fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime failed, status=%s, err=%v\n", m.ID, m.Status, err)
		return
	}
	m.Status = "Online"
	fmt.Fprintf(debugLogFile, "[DEBUG] Machine %d: uptime succeeded, status=%s\n", m.ID, m.Status)
}
