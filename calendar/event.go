package calendar

import (
	"fmt"
	"time"
)

type Event struct {
	ID                string        `json:"EventId"`
	Subject           string        `json:"EventSubject"`
	WebLink           string        `json:"EventWebLink"`
	StartTime         EventTime     `json:"EventStartTime"`
	EndTime           EventTime     `json:"EventEndTime"`
	ReminderFireTime  EventTime     `json:"ReminderFireTime"`
	Location          EventLocation `json:"EventLocation"`
	ReminderSentAt    *time.Time    `json:"-"`
	ReminderSentCount int           `json:"-"`
}

func (evt *Event) String(tz, tf string) string {

	return fmt.Sprintf("%s - %s  %s\n%s", evt.StartTime.TimeLocal(tz).Format(tf),
		evt.EndTime.TimeLocal(tz).Format(tf), evt.Location.DisplayName, evt.WebLink)
}
