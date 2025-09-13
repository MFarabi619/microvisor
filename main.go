package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MachineStatus struct {
    Hostname string
    Chassis  string
    OS       string
    Kernel   string
    GPU      string
    Memory   string
    Display  string
    Uptime   string
    Location string

    Status   string
}

type Model struct {
    width     int
    height    int
    machines  []MachineStatus
    selected  int
    showHelp  bool
    mu        sync.Mutex
}

var (
    titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Background(lipgloss.Color("8")).Padding(0, 1)
    borderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Padding(1, 2)
)

type statusMsg []MachineStatus

func (m *Model) Init() tea.Cmd {
    return fetchStatusesCmd
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "?":
            m.showHelp = true
            return m, nil
        case "esc":
            m.showHelp = false
            return m, nil
        case "up":
            if !m.showHelp && m.selected > 0 {
                m.selected--
            }
            return m, nil
        case "down":
            if !m.showHelp && m.selected < len(m.machines)-1 {
                m.selected++
            }
            return m, nil
        case "enter":
            if !m.showHelp {
                // TODO: logic to select an os
            }
            return m, nil
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    case statusMsg:
    m.mu.Lock()
    m.machines = msg
    m.mu.Unlock()
    return m, fetchStatusesCmd
    }
    return m, nil
}

func truncate(s string, max int) string {
    if len(s) > max {
        return s[:max-1] + "…"
    }
    return s
}

func (m *Model) View() string {
    title := titleStyle.Render(" lazyos ")

    if m.showHelp {
        help := []struct{ key, desc string }{
            {"q", "Quit"},
            {"up/down", "Navigate"},
            {"?", "Show help"},
            {"esc", "Close help"},
        }
        var helpRows []string
        helpRows = append(helpRows, lipgloss.NewStyle().Bold(true).Underline(true).Render("Help Menu"))
        for _, h := range help {
            helpRows = append(helpRows, fmt.Sprintf("%-10s %s", h.key, h.desc))
        }
        helpContent := lipgloss.JoinVertical(lipgloss.Top, helpRows...)
        return borderStyle.Width(m.width).Height(m.height).Render(helpContent)
    }

    colWidths := []int{15, 10, 15, 15, 15, 10, 12, 10, 10, 10}
    headers := []string{"Hostname", "Chassis", "OS", "Kernel", "GPU", "Memory", "Display", "Uptime", "Location", "Status"}
    for i, h := range headers {
        headers[i] = truncate(h, colWidths[i])
    }
    header := fmt.Sprintf("%-15s %-10s %-15s %-15s %-15s %-10s %-12s %-10s %-10s %-10s",
        headers[0], headers[1], headers[2], headers[3], headers[4], headers[5], headers[6], headers[7], headers[8], headers[9])

    var rows []string
    rows = append(rows, lipgloss.NewStyle().Bold(true).Underline(true).Render(header))
    for i, os := range m.machines {
        var statusColor lipgloss.Style
        switch os.Status {
        case "Online":
            statusColor = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
        case "Offline":
            statusColor = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
        default:
            statusColor = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
        }
        fields := []string{
            truncate(os.Hostname, colWidths[0]),
            truncate(os.Chassis, colWidths[1]),
            truncate(os.OS, colWidths[2]),
            truncate(os.Kernel, colWidths[3]),
            truncate(os.GPU, colWidths[4]),
            truncate(os.Memory, colWidths[5]),
            truncate(os.Display, colWidths[6]),
            truncate(os.Uptime, colWidths[7]),
            truncate(os.Location, colWidths[8]),
            truncate(os.Status, colWidths[9]),
        }
        row := fmt.Sprintf("%-15s %-10s %-15s %-15s %-15s %-10s %-12s %-10s %-10s %-10s",
            fields[0], fields[1], fields[2], fields[3], fields[4], fields[5], fields[6], fields[7], fields[8], fields[9])
        row = statusColor.Render(row)
        if i == m.selected {
            row = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Render(row)
        }
        rows = append(rows, row)
    }
    body := lipgloss.JoinVertical(lipgloss.Top, rows...)
    content := lipgloss.JoinVertical(lipgloss.Top, title, "", body)

    border := borderStyle.
        Width(m.width).
        Height(m.height).
        Render(content)

    return border
}

func fetchMachineStatus(cfg *Config) []MachineStatus {
    statuses := make([]MachineStatus, len(cfg.Machines))
    var wg sync.WaitGroup
    for i, m := range cfg.Machines {
        wg.Add(1)
        go func(idx int, mach Machine) {
            defer wg.Done()
            statuses[idx] = MachineStatus{
                Hostname: mach.Hostname,
                Chassis:  "Laptop",
                OS:       "Linux",
                Kernel:   "5.15.0",
                GPU:      "Intel UHD",
                Memory:   "16GB",
                Display:  "1920x1080",
                Uptime:   "2 days",
                Location: "Toronto",
                Status:   "Online",
            }
        }(i, m)
    }
    wg.Wait()
    return statuses
}

func fetchStatusesCmd() tea.Msg {
    cfg, err := LoadConfig("config.yml")
    if err != nil {
        return statusMsg([]MachineStatus{})
    }
    statuses := fetchMachineStatus(cfg)
    time.Sleep(5 * time.Second)
    return statusMsg(statuses)
}

func main() {
    cfg, err := LoadConfig("config.yml")
    if err != nil {
        fmt.Println("Failed to load config:", err)
        os.Exit(1)
    }
    statuses := fetchMachineStatus(cfg)
    model := &Model{
        machines: statuses,
    }
    p := tea.NewProgram(model)
    _, err = p.Run()
    if err != nil {
        fmt.Println("Error running program:", err)
        os.Exit(1)
    }
}