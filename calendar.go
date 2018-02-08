package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type CalDateTime string

func (s CalDateTime) Time() time.Time {

	frontPart := strings.Split(string(s), ".")[0]
	st, err := time.Parse("2006-01-02T15:04:05", frontPart)
	if err != nil {
		log.Fatal(err)
	}
	return st
}

func (s CalDateTime) TimeLocal(z string) time.Time {

	location, _ := time.LoadLocation(*localTZ)
	return s.Time().In(location)
}

type CalEventLocation struct {
	DisplayName string `json:"DisplayName"`
}

type CalEventTime struct {
	DateTime CalDateTime `json:"DateTime"`
	TimeZone string      `json:"TimeZone"`
}

type CalEvent struct {
	EventID           string           `json:"EventId"`
	EventSubject      string           `json:"EventSubject"`
	EventStartTime    CalEventTime     `json:"EventStartTime"`
	EventEndTime      CalEventTime     `json:"EventEndTime"`
	ReminderFireTime  CalEventTime     `json:"ReminderFireTime"`
	EventLocation     CalEventLocation `json:"EventLocation"`
	ReminderSentAt    *time.Time       `json:"-"`
	ReminderSentCount int              `json:"-"`
}

type CalEvents struct {
	Events []CalEvent `json:"value"`
}

type Calendar struct {
	lookAheadInterval time.Duration
	reminders         *CalEvents
	reminderMutex     *sync.Mutex
}

func NewCalendar(lookAheadInterval time.Duration) *Calendar {
	return &Calendar{
		lookAheadInterval: lookAheadInterval,
		reminders: &CalEvents{
			Events: []CalEvent{},
		},
		reminderMutex: &sync.Mutex{},
	}
}

func (evt *CalEvent) String() string {

	return fmt.Sprintf("%s - %s  %s", evt.EventStartTime.DateTime.TimeLocal("America/Los_Angeles").Format(time.Kitchen),
		evt.EventEndTime.DateTime.TimeLocal("America/Los_Angeles").Format(time.Kitchen), evt.EventLocation.DisplayName)
}

func (evt *CalEvent) SendReminder() {

	n := time.Now().UTC()
	exec.Command("notify-send", "-i", *icon, evt.EventSubject, evt.String()).Run()
	evt.ReminderSentAt = &n
	evt.ReminderSentCount++
	if *debug {
		log.Printf("Sent notification for %s\n", evt.String())
	}
}

func (s *Calendar) RemoveEvent(id string) {

	for i := len(s.reminders.Events) - 1; i >= 0; i-- {
		evt := &s.reminders.Events[i]
		if evt.EventID == id {
			s.reminders.Events = append(s.reminders.Events[:i], s.reminders.Events[i+1:]...)
			if *debug {
				log.Println("removed event " + evt.String())
			}
		}
	}
}

func (s *Calendar) AddEvent(evt *CalEvent) {

	found := false
	for i := len(s.reminders.Events) - 1; i >= 0; i-- {
		if s.reminders.Events[i].EventID == evt.EventID {
			found = true
		}
	}
	if !found {
		s.reminders.Events = append(s.reminders.Events, *evt)
		if *debug {
			log.Println("added event " + evt.String())
		}
	}
}

func (s *Calendar) SendReminders() {

	s.reminderMutex.Lock()
	defer s.reminderMutex.Unlock()

	if s.reminders == nil || len(s.reminders.Events) == 0 {
		return
	}

	n := time.Now().UTC()
	for i := 0; i < len(s.reminders.Events); i++ {
		evt := &s.reminders.Events[i]
		if n.After(evt.ReminderFireTime.DateTime.Time()) && evt.ReminderSentCount == 0 {
			// first time, should be ~ ReminderFireTime
			evt.SendReminder()
		} else {
			// send one more, ~ 5 minutes after ReminderFireTime
			if evt.ReminderSentCount == 1 && n.After(evt.ReminderFireTime.DateTime.Time().Add(time.Minute*5)) {
				evt.SendReminder()
				s.RemoveEvent(evt.EventID)
			}
		}
	}
}

func (s *Calendar) RefreshReminders(client *http.Client) error {

	s.reminderMutex.Lock()
	defer s.reminderMutex.Unlock()

	t := time.Now().UTC()
	urlStr := fmt.Sprintf("https://outlook.office.com/api/beta/me/ReminderView(StartDateTime='%s',EndDateTime='%s')",
		t.Format(time.RFC3339), t.Add(s.lookAheadInterval).Format(time.RFC3339))

	resp, err := client.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if *debug {
		log.Println(urlStr)
		log.Println(string(b))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, string(b))
	}

	var cal CalEvents
	err = json.Unmarshal(b, &cal)
	if err != nil {
		return err
	}

	for i := 0; i < len(cal.Events); i++ {
		s.AddEvent(&cal.Events[i])
	}

	return nil
}
