package calsync

import (
	"fmt"
	"net/http"
	"time"

	calendar "google.golang.org/api/calendar/v3"

	"golang.org/x/net/context"
)

// cal implements read and write operations against a google calendar.
type cal struct {
	svc *calendar.Service

	// short name to uniquely identify the application syncing events into
	// a google calendar.
	privateKey string

	// calendar to sync.  If you want to sync into the main calendar, use
	// "primary"
	calID string

	// if this is set, we will will not perform any remove/update/add
	// operations, but will return success, as if we had.
	nop bool
}

func newCal(client *http.Client, privateKey string) (*cal, error) {
	svc, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("failed creating service: %v", err)
	}
	return &cal{
		svc:        svc,
		privateKey: privateKey,
		calID:      "primary"}, nil
}

func (c cal) fetch(ctx context.Context, now time.Time) ([]*Event, error) {
	listResult, err := c.svc.Events.List(c.calID).
		ShowDeleted(false).
		Context(ctx).
		SingleEvents(true).
		TimeMin(now.Format(time.RFC3339)).
		PrivateExtendedProperty(c.privateKey + "=True").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve google calendar events: %v", err)
	}

	idKey := c.idKey()
	var events []*Event
	for _, each := range listResult.Items {
		ev, err := parseEvent(each, idKey)
		if err != nil {
			return nil, fmt.Errorf("parseEvent %q, %v", each.Summary, err)
		}
		events = append(events, ev)
	}

	return events, nil
}

func (c cal) remove(ctx context.Context, ev *Event) error {
	if c.nop {
		return nil
	}
	err := c.svc.Events.Delete(c.calID, ev.calEventID).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("deleting %s: %v", ev.calEventID, err)
	}
	return nil
}

func (c cal) update(ctx context.Context, ev *Event) error {
	if c.nop {
		return nil
	}
	calEvent := c.makeCalEvent(ev)
	_, err := c.svc.Events.Update(c.calID, ev.calEventID, calEvent).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("update %q: %v", ev.Title, err)
	}
	return nil
}

func (c cal) add(ctx context.Context, ev *Event) error {
	if c.nop {
		return nil
	}
	calEvent := c.makeCalEvent(ev)
	_, err := c.svc.Events.Insert(c.calID, calEvent).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("insert %q: %v", ev.Title, err)
	}
	return nil
}

func (c cal) makeCalEvent(ev *Event) *calendar.Event {
	return &calendar.Event{
		Summary:     ev.Title,
		Location:    ev.Where,
		Description: ev.exportedDescription(),

		Start: &calendar.EventDateTime{
			DateTime: ev.Start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: ev.End.Format(time.RFC3339),
		},
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				c.privateKey: "True",
				c.idKey():    ev.SrcID,
			},
		},
	}
}

func (c cal) idKey() string { return c.privateKey + "ID" }
