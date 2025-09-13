package healthcheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseUptime parses the output of the 'uptime' command and returns a readable duration string like "1h25min".
func ParseUptime(uptimeOut string) string {
	// Typical uptime output: " 15:32:10 up  1:25,  2 users,  load average: 0.00, 0.01, 0.05"
	// or " 15:32:10 up  3 days,  1:25,  2 users,  load average: ..."
	// or " 15:32:10 up  1 min,  2 users,  load average: ..."

	re := regexp.MustCompile(`up\s+([^,]+)`)
	match := re.FindStringSubmatch(uptimeOut)
	if len(match) < 2 {
		return "unknown"
	}
	desc := match[1]
	desc = strings.TrimSpace(desc)

	// Handle days
	var days, hours, mins int
	if strings.Contains(desc, "day") {
		parts := strings.Split(desc, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "day") {
				daysStr := strings.Fields(part)[0]
				days, _ = strconv.Atoi(daysStr)
			} else if strings.Contains(part, ":") {
				hm := strings.Split(part, ":")
				if len(hm) == 2 {
					hours, _ = strconv.Atoi(hm[0])
					mins, _ = strconv.Atoi(hm[1])
				}
			}
		}
	} else if strings.Contains(desc, ":") {
		// Format: 1:25
		hm := strings.Split(desc, ":")
		if len(hm) == 2 {
			hours, _ = strconv.Atoi(strings.TrimSpace(hm[0]))
			mins, _ = strconv.Atoi(strings.TrimSpace(hm[1]))
		}
	} else if strings.Contains(desc, "min") {
		// Format: 1 min
		minStr := strings.Fields(desc)[0]
		mins, _ = strconv.Atoi(minStr)
	}

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if len(result) > 0 {
			result += " "
		}
		result += fmt.Sprintf("%dh", hours)
	}
	if mins > 0 {
		if len(result) > 0 {
			result += " "
		}
		result += fmt.Sprintf("%dmin", mins)
	}
	if result == "" {
		result = desc
	}
	return result
}
