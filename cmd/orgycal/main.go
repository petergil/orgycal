package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/apognu/gocal"
	log "github.com/sirupsen/logrus"
)

const (
	orgTimestampFormat = "2006-01-02 Mon 15:04"
)

func main() {
	//log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		//DisableColors: true,
		FullTimestamp: true,
		PadLevelText:  true,
	})

	debug := flag.Bool("debug", false, "enable debug mode")
	inFile := flag.String("in", "cal.ics", "file to read")
	outFile := flag.String("out", "cal.org", "file to write")

	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Setting log level to debug")
	}

	// flag.PrintDefaults()
	cal := getCal(*inFile)

	events := []string{}
	for _, e := range cal.Events {
		events = append(events, orgEntry(e))
	}

	writeOrg(strings.Join(events[:], ""), *outFile)

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
		}).Error("Error reading file")
		panic(err)
	}
	defer f.Close()

	start, end := time.Now().AddDate(0, -6, -1), time.Now().AddDate(1, 1, 0)

	c := gocal.NewParser(f)
	c.Start, c.End = &start, &end
	c.Parse()

	return c
}

func writeOrg(entries string, file string) {
	log.WithFields(log.Fields{
		"file": file,
	}).Debug("Opening org file for writing")
	err := ioutil.WriteFile(file, []byte(entries), 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  file,
			"error": err,
		}).Error("Error writing file")
		panic(err)
	}
}

func orgEntry(event gocal.Event) string {
	t, err := template.New("orgpost").Funcs(
		template.FuncMap{
			"filterDesc": filterDesc,
			"getTags":    getTags,
			//"orgTimeStamp": orgTimeStamp,
			"orgTimeRange": orgTimeRange,
		},
	).Parse(`* {{.Summary}}       {{ getTags . }}
{{- if .Location }}
Location: {{ .Location }}{{- end }}
{{ orgTimeRange .Start .End  true }}

{{filterDesc  .Description }}

`)
	if err != nil {
		log.WithFields(log.Fields{
			"entry": event.Summary,
			"error": err,
		}).Error("Unable to apply template")
		panic(err)
	}

	var entry bytes.Buffer
	t.Execute(&entry, event)
	return entry.String()
}

func filterDesc(description string) string {

	// Microsoft teams meeting boilerplate
	teams, _ := regexp.Compile("_{10,}\\\\n.+<(https://teams.microsoft.com/l/meetup-join/[^ ]+%7d)>.+_{10,}(\\\\n)+")
	// Warnings about external sender
	external, _ := regexp.Compile("EXTERNAL SENDER. Do not click links or open attachments unless you recognize the sender and know the content is safe. DO NOT provide your username or password.\\\\n\\\\n\\\\n")
	// surrounding newlines
	edges, _ := regexp.Compile("(^\\\\n+)|(\\\\n+$)")
	// Consolidate multiple newlines
	newlines, _ := regexp.Compile("(\\\\n)+")

	desc := teams.ReplaceAll([]byte(description), []byte("$1"))
	desc = external.ReplaceAll(desc, []byte(""))
	desc = edges.ReplaceAll(desc, []byte(""))
	desc = newlines.ReplaceAll(desc, []byte("\n"))

	return string(desc)
}

func getTags(event gocal.Event) string {
	tags := []string{}
	teams, _ := regexp.MatchString("Microsoft Teams Meeting", event.Location)
	if teams {
		tags = append(tags, "@teams")
	}
	zoom, _ := regexp.MatchString("https://.+zoom.us/.+", event.Location)
	if zoom {
		tags = append(tags, "@zoom")
	}

	tagString := strings.Join(tags[:], ":")
	if tagString != "" {
		tagString = ":" + tagString + ":"
	}
	return tagString
}

func orgTimeStamp(t *time.Time, active bool) string {
	openchar := "["
	closechar := "]"
	if active {
		openchar = "<"
		closechar = ">"
	}
	return openchar + t.Format(orgTimestampFormat) + closechar
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
