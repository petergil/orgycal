package main

import (
	"testing"
	"time"
)

const tstart = "2023-04-11T11:45:26.371Z"
const tstartTZ = "2023-04-11T10:45:26.371-01:00"
const tend = "2023-04-12T12:45:26.371Z"
const tendTZ = "2023-04-12T13:45:26.371+01:00"
const timerange = "<2023-04-11 Tue 11:45>--<2023-04-12 Wed 12:45>"
const timerangeinactive = "[2023-04-11 Tue 11:45]--[2023-04-12 Wed 12:45]"

func TestTimerange(t *testing.T) {

	// Set local timezone to UTC to work with
	tz = time.UTC

	ts, _ := time.Parse(time.RFC3339, tstart)
	te, _ := time.Parse(time.RFC3339, tend)
	tstz, _ := time.Parse(time.RFC3339, tstartTZ)
	tetz, _ := time.Parse(time.RFC3339, tendTZ)

	tr := orgTimeRange(&ts, &te, true)
	trin := orgTimeRange(&ts, &te, false)
	trtz := orgTimeRange(&tstz, &tetz, true)

	if tr != timerange {
		t.Errorf("'%s' != '%s'", tr, timerange)
	}
	if trin != timerangeinactive {
		t.Errorf("'%s' != '%s' for inactive range", trin, timerangeinactive)
	}
	if trtz != timerange {
		t.Errorf("'%s' != '%s' for timezone-aware range", trtz, timerange)
	}
}

const inviteFile = "testdata/invite.ics"
const inviteResult = `
#+FILETAGS: :foo:bar:

* test invite       :@meet:
<2023-01-30 Mon 09:30>--<2023-01-30 Mon 10:30>

Attendees:
    - ✅ acceptor@example.com
    - ❌ decliner@example.com
    - ⏩ delegator@example.com
    - ⏳ no-action@example.com
    - ❔ undefined@example.com

https://meet.google.com/asd-fghj-klz

`
const fileTags = ":foo:bar:"

func TestFull(t *testing.T) {
	// Set local timezone to UTC to work with
	tz = time.UTC

	ts, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:01.1Z")
	te, _ := time.Parse(time.RFC3339, "2024-01-30T00:00:01.1Z")

	cal := getCal(inviteFile, ts, te)

	contents := orgFormat(cal, fileTags)

	if contents != inviteResult {
		t.Errorf("End-to-end test failed:\n'%s' is not equal to \n'%s'", contents, inviteResult)
	}
}
