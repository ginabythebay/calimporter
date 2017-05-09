package calimporter

import (
	"strings"
	"time"

	"fmt"

	calendar "google.golang.org/api/calendar/v3"
)

const delim = "===================="

type description struct {
	prefix string
	suffix string
}

func parseDescription(s string) *description {
	d := &description{}
	tokens := strings.SplitN(s, delim, 2)
	if len(tokens) == 2 {
		d.prefix = tokens[0]
		d.suffix = tokens[1]
		// In String, below, we insert a newLine between
		// the prefix and the delimiter, and between the delimiter and the
		// suffix.  Strip it back out again here.
		l := len(d.prefix)
		if l != 0 && d.prefix[l-1] == '\n' {
			d.prefix = d.prefix[0 : l-1]
		}
		l = len(d.suffix)
		if l != 0 && d.suffix[0] == '\n' {
			d.suffix = d.suffix[1:]
		}
		return d
	}
	d.suffix = s
	return d
}

func (d *description) String() string {
	if d.prefix == "" {
		return delim + "\n" + d.suffix
	}
	return d.prefix + "\n" + delim + "\n" + d.suffix
}

// Event represents a single importable event.
type Event struct {
	// Title will be used as the summary in google calendar
	Title       string
	Start       time.Time
	End         time.Time
	Where       string
	Description string
	// SrcID will be stored in the google calendar event as a private
	// property and will be used to recognize the same event if you attempt to
	// import it again later.  It should be unique across all events that you
	// import into a single calendar.
	SrcID string

	// only set for events we read from google calendar.  The id assigned by
	// google calendar.
	calEventID string
}

func (ev *Event) String() string {
	return fmt.Sprintf("%s: %s", ev.Start.Format("2006/01/02"), ev.Title)
}

// Has the effect of prepending our delimiter when it is missing.
func (ev *Event) exportedDescription() string {
	d := parseDescription(ev.Description)
	return d.String()
}

func (ev *Event) equal(other *Event) bool {
	if ev.Title != other.Title {
		return false
	}
	if !ev.Start.Equal(other.Start) {
		return false
	}
	if !ev.End.Equal(other.End) {
		return false
	}
	if ev.Where != other.Where {
		return false
	}
	d := parseDescription(ev.Description)
	otherD := parseDescription(other.Description)
	if d.suffix != otherD.suffix {
		return false
	}
	if ev.SrcID != other.SrcID {
		return false
	}
	return true
}

// Returns a new event, which represents an update to ev, based on srcEv.
func (ev *Event) newUpdate(srcEv *Event) *Event {
	update := *srcEv
	update.calEventID = ev.calEventID
	calDescription := parseDescription(ev.Description)
	updateDescription := description{
		prefix: calDescription.prefix,
		suffix: srcEv.Description,
	}
	update.Description = updateDescription.String()
	return &update
}

func parseEvent(in *calendar.Event, idKey string) (*Event, error) {
	title := in.Summary
	start, err := time.Parse(time.RFC3339, in.Start.DateTime)
	if err != nil {
		return nil, fmt.Errorf("unable to parse start %q: %v",
			in.Start.DateTime, err)
	}
	end, err := time.Parse(time.RFC3339, in.End.DateTime)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse end %q: %v",
			in.End.DateTime, err)
	}
	where := in.Location
	description := in.Description

	var props map[string]string
	if in.ExtendedProperties != nil {
		props = in.ExtendedProperties.Private
	}
	srcID := props[idKey]

	return &Event{
		title,
		start,
		end,
		where,
		description,
		srcID,
		in.Id,
	}, nil
}
