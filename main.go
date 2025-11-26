package main

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
)

type Event struct {
	Time          string
	Title         string
	Content       []string
	IsRight       bool // true if contains ❌ or ⁉️
	IsLie         bool // ❌ - proven lie
	IsContradiction bool // ⁉️ - contradiction
	ID            string
}

type Day struct {
	Date       string
	Subtitle   string
	Events     []Event
	ID         string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <timeline-file>")
		os.Exit(1)
	}

	days, title := parseTimeline(os.Args[1])
	generateHTML(days, title)
}

func parseTimeline(filename string) ([]Day, string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var days []Day
	var currentDay *Day
	var currentEvent *Event
	var title string
	idCounts := make(map[string]int) // Track ID occurrences for unique IDs

	scanner := bufio.NewScanner(file)
	lineNum := 0
	dayDateRe := regexp.MustCompile(`^# (\d{4}-\d{2}-\d{2})`)
	timeRe := regexp.MustCompile(`^## (\d{4}-\d{2}-\d{2}[^#]*)$`)
	eventTitleRe := regexp.MustCompile(`^### (.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Skip the main title
		if lineNum == 1 && strings.HasPrefix(line, "# ") && !dayDateRe.MatchString(line) {
			title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Day header
		if matches := dayDateRe.FindStringSubmatch(line); matches != nil {
			// Save current event if any
			if currentEvent != nil && currentDay != nil {
				currentDay.Events = append(currentDay.Events, *currentEvent)
				currentEvent = nil
			}
			// Save current day if any
			if currentDay != nil {
				days = append(days, *currentDay)
			}
			currentDay = &Day{
				Date: matches[1],
				ID:   makeID(matches[1]),
			}
			continue
		}

		// Day subtitle (like "## EL DÍA DE LA CATÁSTROFE")
		if strings.HasPrefix(line, "## ") && !timeRe.MatchString(line) && currentDay != nil && currentEvent == nil {
			currentDay.Subtitle = strings.TrimPrefix(line, "## ")
			continue
		}

		// Time entry
		if matches := timeRe.FindStringSubmatch(line); matches != nil {
			// Save current event if any
			if currentEvent != nil && currentDay != nil {
				currentDay.Events = append(currentDay.Events, *currentEvent)
			}
			baseID := makeID(matches[1])
			idCounts[baseID]++
			eventID := baseID
			if idCounts[baseID] > 1 {
				eventID = fmt.Sprintf("%s-%d", baseID, idCounts[baseID]-1)
			}
			currentEvent = &Event{
				Time: strings.TrimSpace(matches[1]),
				ID:   eventID,
			}
			continue
		}

		// Event title
		if matches := eventTitleRe.FindStringSubmatch(line); matches != nil && currentEvent != nil {
			currentEvent.Title = matches[1]
			// Check if title has lie/contradiction markers
			if strings.Contains(currentEvent.Title, "❌") {
				currentEvent.IsRight = true
				currentEvent.IsLie = true
			}
			if strings.Contains(currentEvent.Title, "⁉️") {
				currentEvent.IsRight = true
				currentEvent.IsContradiction = true
			}
			continue
		}

		// Event content
		if currentEvent != nil && line != "" {
			currentEvent.Content = append(currentEvent.Content, line)
			// Check if this line has lie/contradiction markers
			if strings.Contains(line, "❌") {
				currentEvent.IsRight = true
				currentEvent.IsLie = true
			}
			if strings.Contains(line, "⁉️") {
				currentEvent.IsRight = true
				currentEvent.IsContradiction = true
			}
		}
	}

	// Don't forget the last event and day
	if currentEvent != nil && currentDay != nil {
		currentDay.Events = append(currentDay.Events, *currentEvent)
	}
	if currentDay != nil {
		days = append(days, *currentDay)
	}

	return days, title
}

func makeID(s string) string {
	// Convert to lowercase, replace spaces and special chars with dashes
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "~", "")
	s = strings.ReplaceAll(s, ":", "")
	return s
}

func processContent(content string) string {
	// Escape HTML
	escaped := html.EscapeString(content)

	// Process markdown links [text](url)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	escaped = linkRe.ReplaceAllStringFunc(escaped, func(match string) string {
		// Unescape for processing
		match = strings.ReplaceAll(match, "&amp;", "&")
		match = strings.ReplaceAll(match, "&lt;", "<")
		match = strings.ReplaceAll(match, "&gt;", ">")

		submatches := linkRe.FindStringSubmatch(match)
		if len(submatches) == 3 {
			text := submatches[1]
			url := submatches[2]
			// Internal anchor links don't need target="_blank"
			if strings.HasPrefix(url, "#") {
				return fmt.Sprintf(`<a href="%s" class="text-blue-600 hover:text-blue-800 underline">%s</a>`, url, html.EscapeString(text))
			}
			return fmt.Sprintf(`<a href="%s" class="text-blue-600 hover:text-blue-800 underline" target="_blank" rel="noopener">%s</a>`, url, html.EscapeString(text))
		}
		return match
	})

	// Process bold **text**
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	escaped = boldRe.ReplaceAllString(escaped, `<strong class="font-semibold">$1</strong>`)

	return escaped
}

func generateHTML(days []Day, title string) {
	fmt.Println(`<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + html.EscapeString(title) + `</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .timeline-line {
            position: absolute;
            left: 50%;
            transform: translateX(-50%);
            width: 4px;
            background: linear-gradient(to bottom, #3b82f6, #1d4ed8);
            top: 0;
            bottom: 0;
        }
        .day-dot {
            position: absolute;
            left: 50%;
            transform: translateX(-50%);
            width: 20px;
            height: 20px;
            background: #1d4ed8;
            border: 4px solid #fff;
            border-radius: 50%;
            box-shadow: 0 0 0 2px #1d4ed8;
            z-index: 10;
        }
        .event-left {
            padding-right: 2rem;
            text-align: right;
        }
        .event-right {
            padding-left: 2rem;
            text-align: left;
        }
        .event-card {
            background: white;
            border-radius: 0.5rem;
            padding: 1rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            margin-bottom: 1rem;
        }
        .event-card-lie {
            background: #fef2f2;
            border-left: 4px solid #ef4444;
        }
        .event-card-contradiction {
            background: #fefce8;
            border-left: 4px solid #ef4444;
        }
        .event-card-both {
            background: linear-gradient(135deg, #fef2f2 50%, #fefce8 50%);
            border-left: 4px solid #ef4444;
        }
        .event-card-left {
            background: #f0fdf4;
            border-right: 4px solid #22c55e;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="max-w-7xl mx-auto py-8 px-4">
        <h1 class="text-4xl font-bold text-center mb-12 text-gray-800">` + html.EscapeString(title) + `</h1>

        <div class="flex justify-center gap-8 mb-8">
            <div class="flex items-center gap-2">
                <div class="w-4 h-4 bg-green-500 rounded"></div>
                <span class="text-sm text-gray-600">Hechos</span>
            </div>
            <div class="flex items-center gap-2">
                <div class="w-4 h-4 bg-red-500 rounded"></div>
                <span class="text-sm text-gray-600">Mentiras ❌ / Contradicciones ⁉️</span>
            </div>
        </div>

        <div class="relative">
            <div class="timeline-line"></div>`)

	for _, day := range days {
		fmt.Printf(`
            <!-- Day: %s -->
            <div class="relative mb-8" id="%s">
                <div class="day-dot"></div>
                <div class="text-center py-4">
                    <span class="bg-blue-700 text-white px-4 py-2 rounded-full font-bold text-lg">%s</span>
`, html.EscapeString(day.Date), day.ID, html.EscapeString(day.Date))

		if day.Subtitle != "" {
			fmt.Printf(`                    <div class="mt-2 text-blue-800 font-semibold">%s</div>
`, html.EscapeString(day.Subtitle))
		}

		fmt.Println(`                </div>

                <div class="grid grid-cols-2 gap-0">`)

		for _, event := range day.Events {
			if event.IsRight {
				// Determine card class based on lie/contradiction
				cardClass := "event-card-lie"
				if event.IsLie && event.IsContradiction {
					cardClass = "event-card-both"
				} else if event.IsContradiction {
					cardClass = "event-card-contradiction"
				}
				// Right side: empty left column, content in right column
				fmt.Printf(`                    <div class="col-start-1"></div>
                    <div class="event-right col-start-2" id="%s">
                        <div class="event-card %s">
                            <div class="text-sm text-gray-500 mb-1">%s</div>
                            <div class="font-semibold text-gray-800 mb-2">%s</div>
`, event.ID, html.EscapeString(event.Time), html.EscapeString(event.Title))
			} else {
				// Left side: content in left column, empty right column
				fmt.Printf(`                    <div class="event-left col-start-1" id="%s">
                        <div class="event-card event-card-left">
                            <div class="text-sm text-gray-500 mb-1">%s</div>
                            <div class="font-semibold text-gray-800 mb-2">%s</div>
`, event.ID, html.EscapeString(event.Time), html.EscapeString(event.Title))
			}

			for _, line := range event.Content {
				processed := processContent(line)
				fmt.Printf(`                            <p class="text-sm text-gray-600 mb-1">%s</p>
`, processed)
			}

			if event.IsRight {
				fmt.Println(`                        </div>
                    </div>`)
			} else {
				fmt.Println(`                        </div>
                    </div>
                    <div class="col-start-2"></div>`)
			}
		}

		fmt.Println(`                </div>
            </div>`)
	}

	fmt.Println(`        </div>
    </div>
</body>
</html>`)
}
