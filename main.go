package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xsachax/lazyos/pkg/config"
	"github.com/xsachax/lazyos/pkg/healthcheck"
)

type MachineStatus struct {
    Hostname string
    Chassis  string
    OS       string
    Kernel   string
    CPU      string
    Memory   string
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
    cfg       *config.Config
}

var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("15")).
        Background(lipgloss.Color("57")).
        Padding(0, 2).
        Border(lipgloss.DoubleBorder(), true).
        BorderForeground(lipgloss.Color("57"))

    borderStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("57")).
        Background(lipgloss.Color("236")).
        Padding(1, 3).
        Margin(1, 2).
        Width(0)

    selectedStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("57")).
        Foreground(lipgloss.Color("15")).
        Bold(true).
        Padding(0, 0)

    headerStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("51")).
        Background(lipgloss.Color("236")).
        Padding(0, 1)

    helpKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
    helpDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

type statusMsg []MachineStatus

var debugLogFile *os.File

func init() {
    var err error
    debugLogFile, err = os.OpenFile("lazyos.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        debugLogFile = os.Stderr
    }
}

func (m *Model) fetchStatusesCmd() tea.Cmd {
    return func() tea.Msg {
        statuses := fetchMachineStatus(m.cfg)
        return statusMsg(statuses)
    }
}

func (m *Model) Init() tea.Cmd {
    return m.fetchStatusesCmd()
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
        fmt.Fprintf(debugLogFile, "Update: received statusMsg len=%d\n", len(msg))
        m.mu.Lock()
        m.machines = msg
        m.mu.Unlock()
        // Schedule next update after 5 seconds using tea.Tick
        return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
            return statusMsg(fetchMachineStatus(m.cfg))
        })
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
    title := titleStyle.Render(" 🚀 lazyos ")

    if m.showHelp {
        help := []struct{ key, desc, icon string }{
            {"q", "Quit", "❌"},
            {"up/down", "Navigate", "⬆️⬇️"},
            {"?", "Show help", "❓"},
            {"esc", "Close help", "🔙"},
        }
        var helpRows []string
        helpRows = append(helpRows, lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("51")).Render("🆘 Help Menu"))
        for _, h := range help {
            helpRows = append(helpRows,
                fmt.Sprintf("%s %s %s", helpKeyStyle.Render(h.icon), helpKeyStyle.Render(h.key), helpDescStyle.Render(h.desc)),
            )
        }
    helpContent := lipgloss.JoinVertical(lipgloss.Top, helpRows...)
    return borderStyle.Width(m.width-2).Height(m.height-2).Render(helpContent)
    }

    colWidths := []int{18, 10, 15, 15, 15, 13, 10, 10, 10} // Increased Hostname and Memory widths
    headers := []string{
        "🏷️ Hostname", "🏠 Chassis", "🖥️ OS", "🧬 Kernel", "🧠 CPU", "💾 Memory", "⏳ Uptime", "📍 Location", "Status"}
    // Do not truncate headers with icons
    header := headerStyle.Render(fmt.Sprintf("%-18s %-10s %-15s %-15s %-15s %-13s %-10s %-10s %-10s",
        headers[0], headers[1], headers[2], headers[3], headers[4], headers[5], headers[6], headers[7], headers[8]))

    var rows []string
    rows = append(rows, header)
    for i, os := range m.machines {
        var statusIcon, statusColor string
        switch os.Status {
        case "Online":
            statusIcon = "🟢"
            statusColor = "2"
        case "Offline":
            statusIcon = "🔴"
            statusColor = "1"
        default:
            statusIcon = "⚪️"
            statusColor = "8"
        }
        fields := make([]string, len(colWidths))
        fields[0] = pad(truncate(os.Hostname, colWidths[0]), colWidths[0])
        fields[1] = pad(truncate(trim(firstLine(os.Chassis)), colWidths[1]), colWidths[1])
        fields[2] = pad(truncate(os.OS, colWidths[2]), colWidths[2])
        fields[3] = pad(truncate(os.Kernel, colWidths[3]), colWidths[3])
        fields[4] = pad(truncate(firstLine(os.CPU), colWidths[4]), colWidths[4])
        fields[5] = pad(truncate(firstLine(os.Memory), colWidths[5]), colWidths[5])
        fields[6] = pad(truncate(os.Uptime, colWidths[6]), colWidths[6])
        fields[7] = pad(truncate(os.Location, colWidths[7]), colWidths[7])
        // Do not pad the last column
        fields[8] = truncate(os.Status, colWidths[8])
        row := fmt.Sprintf("%s %s", statusIcon, strings.Join(fields, " "))
        row = strings.TrimRight(row, " ")
        styledRow := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(row)
        if i == m.selected {
            styledRow = selectedStyle.Render(row)
        }
        rows = append(rows, styledRow)
    }
    body := lipgloss.JoinVertical(lipgloss.Top, rows...)
    content := lipgloss.JoinVertical(lipgloss.Top, title, "", body)

    border := borderStyle.
        Width(m.width-2).
        Height(m.height-2).
        Render(content)

    return border
}

func pad(s string, width int) string {
    if len(s) < width {
        return s + strings.Repeat(" ", width-len(s))
    }
    return s
}

func fetchMachineStatus(cfg *config.Config) []MachineStatus {
    infos := healthcheck.ConvertMachines(cfg.Machines)
    statuses := make([]MachineStatus, len(infos))
    var wg sync.WaitGroup
    for i, info := range infos {
        wg.Add(1)
        go func(idx int, inf healthcheck.MachineInfo) {
            defer wg.Done()
            healthcheck.FetchStatus(&inf)
            chassis := strings.TrimSpace(inf.Chassis)
            if strings.HasPrefix(chassis, "Chassis:") {
                chassis = strings.TrimSpace(strings.TrimPrefix(chassis, "Chassis:"))
                // Only take the first word after 'Chassis:'
                if len(chassis) > 0 {
                    chassis = strings.Fields(chassis)[0]
                }
            }
            // Only show city name in location
            statuses[idx] = MachineStatus{
                Hostname: inf.Hostname,
                Status:   inf.Status,
                Uptime:   inf.Uptime,
                OS:       inf.OS,
                Kernel:   inf.Kernel,
                CPU:      inf.CPU,
                Memory:   inf.Memory,
                Chassis:  chassis,
            }
        }(i, info)
    }
    wg.Wait()
    return statuses
}

func firstLine(s string) string {
    for i, c := range s {
        if c == '\n' {
            return s[:i]
        }
    }
    return s
}

func trim(s string) string {
    return strings.TrimSpace(s)
}

func main() {
    // Clear log file before each execution
    _ = os.WriteFile("lazyos.log", []byte{}, 0644)
    cfg, err := config.LoadConfig("config.yml")
    if err != nil {
        fmt.Println("Failed to load config:", err)
        os.Exit(1)
    }
    model := &Model{cfg: cfg}
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err = p.Run()
    if err != nil {
        fmt.Println("Error running program:", err)
        os.Exit(1)
    }
}