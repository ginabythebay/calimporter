package calimporter

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGetOperations(t *testing.T) {
	now := when("2017-04-29T20:00:00-07:00")

	// This will be the same in fetlife and in calendar.  Should not
	// be part of any operation.
	same := newSrcEvent("same", now.Add(time.Hour))

	// This will exist in both fetlife in calendar, but will be
	// different and we expect it to be part of an update operation.
	changed := newSrcEvent("changed", now.AddDate(0, 0, 1))

	// Exists in both fetlife and calendar.  The only difference is
	// that the calendar version has a prefix comment.  We do not
	// expect it to be part of any operation.
	hasComment := newSrcEvent("hasComment", now.AddDate(0, 0, 2))

	// Exists in both fetlife and calendar.  It has a calendar comment
	// and additionally has changed.  We expect it to appear in the
	// updates and we expect the comment to be maintained.
	hasCommentWithChange := newSrcEvent("hasCommentWithChange", now.AddDate(0, 0, 2))

	// Exists in fetlife but not in calendar.  Expected in add operation.
	newEvent := newSrcEvent("newEvent", now.AddDate(0, 0, 3))

	// Exists in calendar but not in fetlife.  Expected in delete operation.
	removedEvent := newSrcEvent("removedEvent", now.AddDate(0, 0, 4))

	srcEvents := []*Event{
		same,
		changed,
		hasComment,
		hasCommentWithChange,
		newEvent,
	}

	calEvents := []*Event{
		testCalEvent("", "", same),
		testCalEvent("", "This is a change", changed),
		testCalEvent("This is a comment", "", hasComment),
		testCalEvent("This is a comment", "ThisIsAChange", hasCommentWithChange),
		testCalEvent("", "", removedEvent),
	}

	changes := getOperations(now, calEvents, srcEvents)

	equals(t, 1, len(changes.Deletes))
	equals(t, "removedEvent title", changes.Deletes[0].calEventID)

	equals(t, 2, len(changes.Updates))
	findEvent(t, "changed title", changes.Updates)
	ev := findEvent(t, "hasCommentWithChange title", changes.Updates)
	assert(t, strings.HasPrefix(ev.Description, "This is a comment\n"+delim), "unexpected description %s", ev.Description)

	equals(t, 1, len(changes.Adds))
	equals(t, "newEvent title", changes.Adds[0].Title)
}

func findEvent(tb testing.TB, title string, events []*Event) *Event {
	for _, ev := range events {
		if ev.Title == title {
			return ev
		}
	}
	tb.Errorf("Unable to find %s", title)
	tb.FailNow()
	return nil
}

func testCalEvent(prefix string, suffix string, srcEvent *Event) *Event {
	calEvent := *srcEvent
	desc := description{
		prefix: prefix,
		suffix: srcEvent.Description,
	}
	if suffix != "" {
		desc.suffix = desc.suffix + "\n" + suffix
	}
	calEvent.Description = desc.String()
	calEvent.calEventID = srcEvent.Title
	return &calEvent
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

func when(s string) time.Time {
	ret, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return ret
}

func cat(s ...string) string {
	return strings.Join(s, " ")
}

func newSrcEvent(name string, start time.Time) *Event {
	return &Event{
		Title:       cat(name, "title"),
		Start:       start,
		End:         start.Add(time.Hour),
		Where:       cat(name, "where"),
		Description: cat(name, "description"),
		SrcID:       cat(name, "srcId"),
	}
}
