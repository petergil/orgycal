package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/apognu/gocal"
	log "github.com/sirupsen/logrus"
)

const (
	orgTimestampFormat = "2006-01-02 Mon 15:04"
)

type OrgMeta struct {
	FileTags string
}

// rfc5545 section-3.2.12 partstat-event to emojis
var partStatusMap = map[string]string{
	"ACCEPTED":     "✅",
	"DECLINED":     "❌",
	"TENTATIVE":    "❓",
	"DELEGATED":    "⏩",
	"NEEDS-ACTION": "⏳", // Waiting for reply
}

var tz *time.Location = time.Local

func main() {
	//log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		//DisableColors: true,
		FullTimestamp: true,
		PadLevelText:  true,
	})

	debug := flag.Bool("debug", false, "enable debug mode")
	inFile := flag.String("in", "cal.ics", "file to read")
	outFile := flag.String("out", "cal.org", "file to write ('-' means stdout)")
	tags := flag.String("tags", ":orgycal:", "add the following filetags to the generated file (:-separated list)")
	timezone := flag.String("timezone", "local", "which timezone to output timestamps in ('local' tries to extract the current user timezone)")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Setting log level to debug")
	}

	log.WithFields(log.Fields{
		"timezone": *timezone}).Debug("Setting timezone to")
	if *timezone != "local" {
		var err error
		tz, err = time.LoadLocation(*timezone)
		if err != nil {
			log.WithFields(log.Fields{
				"timezone": *timezone,
				"error":    err,
			}).Fatal("Unable to load timezone")
		}
	}

	// flag.PrintDefaults()
	cal := getCal(*inFile)

	events := []string{}
	for _, e := range cal.Events {
		events = append(events, orgEntry(e))
	}

	meta := OrgMeta{FileTags: *tags}

	contents := orgHeader(meta) + strings.Join(events[:], "")

	if *outFile == "-" {
		fmt.Println(contents)
	} else {
		writeOrg(contents, *outFile)
	}
}

func getCal(file string) *gocal.Gocal {
	log.WithFields(log.Fields{
		"file": file,
	}).Debug("Opening calendar file")
	f, err := os.Open(file)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  file,
			"error": err,
		}).Fatal("Error reading file")
	}
	defer f.Close()

	start, end := time.Now().AddDate(0, -6, -1), time.Now().AddDate(1, 1, 0)

	c := gocal.NewParser(f)
	c.Start, c.End = &start, &end
	if err := c.Parse(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to parse input file")
	}

	return c
}

func writeOrg(entries string, file string) {
	log.WithFields(log.Fields{
		"file": file,
	}).Debug("Opening org file for writing")
	err := os.WriteFile(file, []byte(entries), 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  file,
			"error": err,
		}).Fatal("Error writing file")
	}
}

func orgHeader(meta OrgMeta) string {
	h, _ := template.New("orgheader").Parse(`
#+FILETAGS: {{ .FileTags}}

`)

	var header bytes.Buffer
	if err := h.Execute(&header, meta); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to apply header")
	}
	log.WithFields(log.Fields{
		"meta": meta,
	}).Debug("Formatted header")

	return header.String()

}

// map participation status to string representation
func orgAttendeeStatus(status string) string {
	ret := partStatusMap[status]
	if ret == "" {
		ret = "❔"
	}
	return ret
}

func orgAttendee(attendee gocal.Attendee) string {
	return "  - " + orgAttendeeStatus(attendee.Status) + " " + attendee.Cn
}

func orgAttendees(attendees []gocal.Attendee) string {
	ret := "Attendees: "

	keys := make([]string, 0, len(attendees))
	att := map[string]string{}
	for _, a := range attendees {
		att[a.Cn] = orgAttendee(a)
		keys = append(keys, a.Cn)
	}
	sort.Strings(keys)
	for _, a := range keys {
		ret += "\n  " + att[a]
	}

	return ret
}

func orgEntry(event gocal.Event) string {
	t, err := template.New("orgpost").Funcs(
		template.FuncMap{
			"filterDesc":      filterDesc,
			"formatAttendees": orgAttendees,
			"getTags":         getTags,
			//"orgTimeStamp": orgTimeStamp,
			"orgTimeRange": orgTimeRange,
		},
	).Parse(`* {{.Summary}}       {{ getTags . }}
{{- if .Location }}
Location: {{ .Location }}{{- end }}
{{ orgTimeRange .Start .End  true }}{{- if .Attendees}}

{{formatAttendees .Attendees }}{{- end }}

{{filterDesc  .Description }}

`)
	if err != nil {
		log.WithFields(log.Fields{
			"entry": event.Summary,
			"error": err,
		}).Fatal("Unable to parse template") // TODO: should this be fatal or not?
	}

	var entry bytes.Buffer
	if err := t.Execute(&entry, event); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to apply entry")
	}
	log.WithFields(log.Fields{
		"entrystart": event.Start,
		"entry":      event.Summary,
	}).Debug("Formatted entry")

	return entry.String()
}

func filterDesc(description string) string {

	descriptionFilters := [][]string{
		// Microsoft teams boilerplate
		{"_{10,}\\\\n.+<(https://teams.microsoft.com/l/meetup-join/[^ ]+%7d)>.+_{10,}(\\\\n)+", "$1"},
		// Google meet boilerplate
		{"^.+ Google Meet: (https://meet.google.com/[a-z-]+)\\\\n.+https://support.google.com/a/users/answer/9282720", "$1"},
		// Outlook warning about external sender
		{"EXTERNAL SENDER. Do not click links or open attachments unless you recognize the sender and know the content is safe. DO NOT provide your username or password.\\\\n\\\\n\\\\n", ""},
		// surrounding newlines
		{"(^\\\\n+)|(\\\\n+$)", ""},
		// Consolidate multiple newlines
		{"(\\\\n)+", "\n"},
	}

	desc := []byte(description)

	for _, pattern := range descriptionFilters {
		regex, _ := regexp.Compile(pattern[0])
		desc = regex.ReplaceAll(desc, []byte(pattern[1]))
	}

	return string(desc)
}

func getTags(event gocal.Event) string {
	tags := []string{}

	locationTags := [][]string{
		{"Microsoft Teams Meeting", "@teams"},
		{"https://.+zoom.us/.+", "@zoom"},
		{"https://meet.google.com/[a-z-]+", "@meet"},
	}

	descriptionTags := [][]string{
		{"https://teams.microsoft.com/l/meetup-join/", "@teams"},
		{"https://meet.google.com/[a-z-]+", "@meet"},
	}

	for _, pattern := range locationTags {
		tagMatch, _ := regexp.MatchString(pattern[0], event.Location)
		if tagMatch && !contains(tags, pattern[1]) {
			tags = append(tags, pattern[1])
		}
	}

	for _, pattern := range descriptionTags {
		tagMatch, _ := regexp.MatchString(pattern[0], event.Description)
		if tagMatch && !contains(tags, pattern[1]) {
			tags = append(tags, pattern[1])
		}
	}

	sort.Strings(tags)
	tagString := strings.Join(tags[:], ":")
	if tagString != "" {
		tagString = ":" + tagString + ":"
	}
	return tagString
}

func contains(list []string, value string) bool {
	for _, ii := range list {
		if ii == value {
			return true
		}
	}
	return false
}

func orgTimeStamp(t *time.Time, active bool) string {
	openchar := "["
	closechar := "]"
	if active {
		openchar = "<"
		closechar = ">"
	}

	return openchar + t.In(tz).Format(orgTimestampFormat) + closechar
}

func orgTimeRange(start *time.Time, end *time.Time, active bool) string {
	separator := "--"
	return orgTimeStamp(start, active) + separator + orgTimeStamp(end, active)
}

// Fancy version that can produce short-style timestamps
// func orgTimeRange(start *time.Time, end *time.Time, active bool) string{
// 	// is this an active or inactive timestamp
// 	openchar := "["
// 	closechar := "]"
// 	if active {
// 		openchar = "<"
// 		closechar = ">"
// 	}
// 	startStr := start.Format(openchar + orgTimestampFormat)
// 	separator := closechar + "--" + openchar
// 	endStr := start.Format(orgTimestampFormat + closechar)
// 	// Check if we can use short time ranges
// 	if start.Truncate(24*time.Hour).Equal(end.Truncate(24*time.Hour)){
// 		separator = "--"
// 		endStr = end.Format("15:04" + closechar)
// 	}
// 	return startStr + separator + endStr
// }
