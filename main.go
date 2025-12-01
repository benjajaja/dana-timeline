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
	Time            string
	Title           string
	Content         []string
	IsRight         bool // true if contains ❌ or ⁉️
	IsLie           bool // ❌ - proven lie
	IsContradiction bool // ⁉️ - contradiction
	IsDark          bool // between --- markers
	ID              string
}

type Day struct {
	Date         string
	Subtitle     string
	SectionTitle string // For non-date # headers like "Las horas desaparecidas"
	Events       []Event
	ID           string
	IsDark       bool
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
	inDarkSection := false           // Track if we're between --- markers

	scanner := bufio.NewScanner(file)
	lineNum := 0
	dayDateRe := regexp.MustCompile(`^# (\d{4}-\d{2}-\d{2})`)
	timeRe := regexp.MustCompile(`^## (\d{4}-\d{2}-\d{2}[^#]*)$`)
	eventTitleRe := regexp.MustCompile(`^### (.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Toggle dark section on --- markers
		if line == "---" {
			inDarkSection = !inDarkSection
			continue
		}

		// Skip the main title
		if lineNum == 1 && strings.HasPrefix(line, "# ") && !dayDateRe.MatchString(line) {
			title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Day header (date format)
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
				Date:   matches[1],
				ID:     makeID(matches[1]),
				IsDark: inDarkSection,
			}
			continue
		}

		// Section header (non-date # header like "# Las horas desaparecidas")
		if strings.HasPrefix(line, "# ") && !dayDateRe.MatchString(line) && lineNum > 1 {
			// Save current event if any
			if currentEvent != nil && currentDay != nil {
				currentDay.Events = append(currentDay.Events, *currentEvent)
				currentEvent = nil
			}
			// Save current day if any
			if currentDay != nil {
				days = append(days, *currentDay)
			}
			sectionTitle := strings.TrimPrefix(line, "# ")
			currentDay = &Day{
				SectionTitle: sectionTitle,
				ID:           makeID(sectionTitle),
				IsDark:       inDarkSection,
			}
			continue
		}

		// Day subtitle (like "## EL DÍA DE LA CATÁSTROFE")
		if strings.HasPrefix(line, "## ") && !timeRe.MatchString(line) && currentDay != nil &&
			currentEvent == nil {
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
				Time:   strings.TrimSpace(matches[1]),
				ID:     eventID,
				IsDark: inDarkSection,
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

	// Process markdown images ![alt](url) - must be before links
	imgRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	escaped = imgRe.ReplaceAllStringFunc(escaped, func(match string) string {
		// Unescape for processing
		match = strings.ReplaceAll(match, "&amp;", "&")
		match = strings.ReplaceAll(match, "&lt;", "<")
		match = strings.ReplaceAll(match, "&gt;", ">")

		submatches := imgRe.FindStringSubmatch(match)
		if len(submatches) == 3 {
			alt := submatches[1]
			url := submatches[2]
			// Check if it's an mp4 file - render as video player
			if strings.HasSuffix(strings.ToLower(url), ".mp4") {
				return fmt.Sprintf(
					`<video controls class="max-w-full h-auto rounded my-2"><source src="%s" type="video/mp4">%s</video>`,
					url,
					html.EscapeString(alt),
				)
			}
			return fmt.Sprintf(
				`<img src="%s" alt="%s" class="max-w-full h-auto rounded my-2">`,
				url,
				html.EscapeString(alt),
			)
		}
		return match
	})

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
				return fmt.Sprintf(
					`<a href="%s" class="text-blue-600 hover:text-blue-800 underline">%s</a>`,
					url,
					html.EscapeString(text),
				)
			}
			return fmt.Sprintf(
				`<a href="%s" class="text-blue-600 hover:text-blue-800 underline" target="_blank" rel="noopener">%s</a>`,
				url,
				html.EscapeString(text),
			)
		}
		return match
	})

	// Process bold **text**
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	escaped = boldRe.ReplaceAllString(escaped, `<strong class="font-semibold">$1</strong>`)

	// Process blockquotes (lines starting with >)
	// After HTML escaping, > becomes &gt;
	if strings.HasPrefix(escaped, "&gt;") {
		inner := strings.TrimPrefix(escaped, "&gt;")
		inner = strings.TrimPrefix(inner, " ") // optional space after >
		escaped = fmt.Sprintf(
			`<blockquote class="border-l-4 border-gray-300 pl-3 ml-2 text-gray-600">%s</blockquote>`,
			inner,
		)
	}

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
            z-index: 0;
        }
        .events-container {
            display: flex;
            position: relative;
        }
        .events-col-left {
            width: calc(50% - 50px);
            padding-right: 0;
        }
        .events-col-center {
            width: 100px;
            position: relative;
            flex-shrink: 0;
        }
        .events-col-right {
            width: calc(50% - 50px);
            padding-left: 0;
        }
        .event-wrapper {
            position: relative;
            margin-bottom: 1rem;
        }
        .event-wrapper-left {
            padding-right: 1rem;
        }
        .event-wrapper-right {
            padding-left: 1rem;
        }
        .timestamp {
            position: absolute;
            background: white;
            border: 2px solid #3b82f6;
            border-radius: 0.5rem;
            padding: 0 0.5rem;
            font-size: 0.75rem;
            font-weight: 600;
            color: #1d4ed8;
            text-align: center;
            white-space: nowrap;
            top: 0.75rem;
        }
        .timestamp::before {
            content: '';
            position: absolute;
            top: 50%;
            height: 2px;
            background: #3b82f6;
            z-index: -100;
        }
        .timestamp-left {
            right: -50px;
            transform: translateX(50%);
        }
        .timestamp-left::before {
            right: 100%;
            width: 200px;
        }
        .timestamp-right {
            left: -50px;
            transform: translateX(-50%);
        }
        .timestamp-right::before {
            left: 100%;
            width: 200px;
        }
        .timestamp-dark {
            background: #1e293b;
            border-color: #64748b;
            color: #e2e8f0;
        }
        .event-card {
            background: white;
            border-radius: 0.5rem;
            padding: 1rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            position: relative;
            z-index: 1;
        }
        .event-card-lie {
            background: #fef2f2;
            border-right: 4px solid #ef4444;
        }
        .event-card-contradiction {
            background: #fefce8;
            border-right: 4px solid #ef4444;
        }
        .event-card-both {
            background: linear-gradient(135deg, #fef2f2 50%, #fefce8 50%);
            border-right: 4px solid #ef4444;
        }
        .event-card-left {
            background: #f0fdf4;
            border-left: 4px solid #22c55e;
        }
        .dark-section {
            background: rgba(0, 0, 0, 0.5);
            margin-left: -1rem;
            margin-right: -1rem;
            padding-left: 1rem;
            padding-right: 1rem;
            padding-top: 1rem;
            padding-bottom: 1rem;
        }
        @media (max-width: 800px) {
            .events-container {
                flex-direction: column;
            }
            .events-col-left, .events-col-right {
                width: 100%;
            }
            .events-col-center {
                display: none;
            }
            .event-wrapper-left, .event-wrapper-right {
                padding: 0;
            }
            .timestamp {
                position: relative;
                top: 0;
                left: 0;
                right: auto;
                transform: none;
                margin-bottom: 0.5rem;
                display: inline-block;
            }
            .timestamp::before {
                display: none;
            }
            .timeline-line {
                display: none;
            }
            .text-center.my-4 {
                text-align: left;
            }
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="max-w-7xl mx-auto py-8 px-4">
        <h1 class="text-4xl font-bold text-center mb-12 text-gray-800">` + html.EscapeString(title) + `</h1>

        <!--<div class="flex justify-center flex-wrap gap-1 mb-8">
            <div class="bg-red-50 border-r-4 border-red-500 rounded p-1">
                <span class="text-xs text-gray-700">❌ Mentiras</span>
            </div>
            <div class="bg-yellow-50 border-r-4 border-red-500 rounded p-1">
                <span class="text-xs text-gray-700">⁉️ Contradicciones</span>
            </div>
            <div class="bg-green-100 border-l-4 border-green-500 rounded p-1">
                <span class="text-xs text-gray-700">Hechos</span>
            </div>
        </div>-->

        <div class="relative">
            <div class="timeline-line"></div>`)

	for _, day := range days {
		if day.SectionTitle != "" {
			// Section header (non-date, like "Las horas desaparecidas")
			fmt.Printf(`
            <!-- Section: %s -->
            <div class="relative mb-8" id="%s">
                <div class="text-center my-4">
                    <span class="bg-gray-800 text-white px-2 py-2 rounded-full font-bold text-lg">%s</span>
`, html.EscapeString(day.SectionTitle), day.ID, html.EscapeString(day.SectionTitle))
		} else {
			// Regular day header
			fmt.Printf(`
            <!-- Day: %s -->
            <div class="relative mb-8" id="%s">
                <div class="text-center my-4">
                    <span class="bg-blue-700 text-white px-4 py-2 rounded-full font-bold text-lg">%s</span>
`, html.EscapeString(day.Date), day.ID, html.EscapeString(day.Date))

			if day.Subtitle != "" {
				fmt.Printf(`                    <div class="mt-2 text-blue-800 font-semibold">%s</div>
`, html.EscapeString(day.Subtitle))
			}
		}

		fmt.Println(`                </div>
                <div class="events-container">
                    <div class="events-col-left">`)

		// Output left column (facts)
		inDarkOutput := false
		for _, event := range day.Events {
			if event.IsRight {
				continue
			}
			if event.IsDark && !inDarkOutput {
				fmt.Println(`                        <div class="dark-section">`)
				inDarkOutput = true
			} else if !event.IsDark && inDarkOutput {
				fmt.Println(`                        </div>`)
				inDarkOutput = false
			}

			timePart := event.Time
			if len(event.Time) > 10 {
				timePart = strings.TrimSpace(event.Time[10:])
			}
			darkClass := ""
			if event.IsDark {
				darkClass = " timestamp-dark"
			}

			fmt.Printf(
				`                        <div class="event-wrapper event-wrapper-left" id="%s">
                            <div class="timestamp timestamp-left%s">%s</div>
                            <div class="event-card event-card-left">
                                <div class="font-semibold text-gray-800 mb-2">%s</div>
`,
				event.ID,
				darkClass,
				html.EscapeString(timePart),
				html.EscapeString(event.Title),
			)
			for _, line := range event.Content {
				processed := processContent(line)
				fmt.Printf(
					`                                <p class="text-sm text-gray-600 mb-1">%s</p>
`,
					processed,
				)
			}
			fmt.Println(`                            </div>
                        </div>`)
		}
		if inDarkOutput {
			fmt.Println(`                        </div>`)
		}

		fmt.Println(`                    </div>
                    <div class="events-col-center"></div>
                    <div class="events-col-right">`)

		// Output right column (lies/contradictions)
		inDarkOutput = false
		for _, event := range day.Events {
			if !event.IsRight {
				continue
			}
			if event.IsDark && !inDarkOutput {
				fmt.Println(`                        <div class="dark-section">`)
				inDarkOutput = true
			} else if !event.IsDark && inDarkOutput {
				fmt.Println(`                        </div>`)
				inDarkOutput = false
			}

			timePart := event.Time
			if len(event.Time) > 10 {
				timePart = strings.TrimSpace(event.Time[10:])
			}
			darkClass := ""
			if event.IsDark {
				darkClass = " timestamp-dark"
			}

			cardClass := "event-card-lie"
			if event.IsLie && event.IsContradiction {
				cardClass = "event-card-both"
			} else if event.IsContradiction {
				cardClass = "event-card-contradiction"
			}
			fmt.Printf(
				`                        <div class="event-wrapper event-wrapper-right" id="%s">
                            <div class="timestamp timestamp-right%s">%s</div>
                            <div class="event-card %s">
                                <div class="font-semibold text-gray-800 mb-2">%s</div>
`,
				event.ID,
				darkClass,
				html.EscapeString(timePart),
				cardClass,
				html.EscapeString(event.Title),
			)
			for _, line := range event.Content {
				processed := processContent(line)
				fmt.Printf(
					`                                <p class="text-sm text-gray-600 mb-1">%s</p>
`,
					processed,
				)
			}
			fmt.Println(`                            </div>
                        </div>`)
		}
		if inDarkOutput {
			fmt.Println(`                        </div>`)
		}

		fmt.Println(`                    </div>
                </div>`)
		fmt.Println(`            </div>`)
	}

	fmt.Println(`        </div>
    </div>
</body>
</html>`)
}
