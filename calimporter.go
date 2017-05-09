/*
Package calimporter helps import events into a Google Calendar.

Importing the same set of events into a google calendar a second time
will have no effect if the events have not been modified in google
calendar.  If the events have been modified in google calendar and
then are imported again, they will be overwritten, in general.

All imported events will start with a delimiter string:

    ====================

Users of google calendar may put any text they like before the
delimiter string and this package will maintain that text of the event
during any subsequent updates.

We use google calendar private extended properties to store data that
lets us re-import safely.  Each created event will have a private
extended property of the form <privateKey>=True and another one of
the form <privateKey>ID=<srcID>.  The first private property allows
us to query for all matching events in subsequent imports.  The second
private propery lets us match up srcEvents with google calendar events
in subsequent imports so we can properly add/update/delete as
appropriate.
*/
package calimporter

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	calendar "google.golang.org/api/calendar/v3"

	"golang.org/x/net/context"
)

// Scope is the scope we need to read and write calendars.
const Scope = calendar.CalendarScope

// Changes represents a set of changes that were made as the result of
// an Import call.
type Changes struct {
	Deletes, Updates, Adds []*Event
}

func (c *Changes) String() string {
	var lines []string
	for _, ev := range c.Deletes {
		lines = append(lines, fmt.Sprintf("Delete %s", ev))
	}
	for _, ev := range c.Updates {
		lines = append(lines, fmt.Sprintf("Update %s", ev))
	}
	for _, ev := range c.Adds {
		lines = append(lines, fmt.Sprintf("Add %s", ev))
	}
	return strings.Join(lines, "\n")
}

// Import imports srcEvents into a google calendar.  See the package
// comments for more details.
func Import(
	ctx context.Context,
	client *http.Client,
	privateKey string,
	srcEvents []*Event,
	opts ...Opt) (*Changes, error) {
	now := time.Now()

	c, err := newCal(client, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed creating cal: %v", err)
	}
	for _, o := range opts {
		o(c)
	}

	calEvents, err := c.fetch(ctx, now)

	changes := getOperations(now, calEvents, srcEvents)
	for _, ev := range changes.Deletes {
		if err = c.remove(ctx, ev); err != nil {
			return nil, err
		}
	}

	for _, u := range changes.Updates {
		if err = c.update(ctx, u); err != nil {
			return nil, err
		}
	}

	for _, ev := range changes.Adds {
		if err := c.add(ctx, ev); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

func getOperations(now time.Time, calEvents, srcEvents []*Event) *Changes {
	changes := Changes{}

	srcMap := map[string]*Event{}
	for _, ev := range srcEvents {
		if ev.End.Before(now) {
			continue
		}
		srcMap[ev.SrcID] = ev
	}

	for _, calEv := range calEvents {
		srcEv, ok := srcMap[calEv.SrcID]
		if ok {
			if !srcEv.equal(calEv) {
				changes.Updates = append(changes.Updates, calEv.newUpdate(srcEv))
			}
			delete(srcMap, calEv.SrcID)
		} else {
			changes.Deletes = append(changes.Deletes, calEv)
		}
	}

	for _, srcEv := range srcMap {
		changes.Adds = append(changes.Adds, srcEv)
	}

	return &changes
}

// Opt is an optional way to configure the Import command.
type Opt func(c *cal)

// CalendarID will override the default of accessing the users primary
// calendar, instead accessing the calendar identified by calID.
func CalendarID(calID string) Opt {
	return func(c *cal) {
		c.calID = calID
	}
}

// Nop make the Import call operation in readonly mode, reporting what
// it would have done without modifying anything.
func Nop() Opt {
	return func(c *cal) {
		c.nop = true
	}
}
