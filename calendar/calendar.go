package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type Calendar struct {
	oauth2Config    *oauth2.Config
	tickInterval    time.Duration
	refreshInterval time.Duration
	cacheDuration   time.Duration
	tz              string
	ch              chan<- *Event
	eventCache      map[string]*Event
	ticker          *time.Ticker
}

func New(oauth2Config *oauth2.Config, tickInterval time.Duration, refreshInterval time.Duration, cacheDuration time.Duration, tz string, ch chan<- *Event) *Calendar {

	return &Calendar{
		oauth2Config:    oauth2Config,
		tickInterval:    tickInterval,
		refreshInterval: refreshInterval,
		cacheDuration:   cacheDuration,
		tz:              tz,
		ch:              ch,
		eventCache:      map[string]*Event{},
	}
}

func (s *Calendar) Start(ctx context.Context, token *oauth2.Token) {

	lastRefresh := time.Now()
	var toDelete []string
	var k string
	var v *Event
	var i int
	var err error

	s.ticker = time.NewTicker(s.tickInterval)
	log.Printf("starting ticker with interval of %s\n", s.tickInterval.String())

	// go ahead and grab first batch
	err = s.getEvents(ctx, token, lastRefresh, s.cacheDuration)
	if err != nil {
		log.Println(err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("exiting ticker")
			s.ticker.Stop()
			return
		case <-s.ticker.C:
			// see if we need to refresh the event cache
			if time.Since(lastRefresh) > s.refreshInterval {
				lastRefresh = time.Now()
				err = s.getEvents(ctx, token, lastRefresh, s.cacheDuration)
				if err != nil {
					log.Println(err.Error())
				}
			}
			// check event cache for things to send
			toDelete = []string{}
			for k, v = range s.eventCache {
				if v.ReminderFireTime.TimeLocal(s.tz).Before(time.Now()) {
					s.ch <- v
					toDelete = append(toDelete, k)
				}
			}
			for i = 0; i < len(toDelete); i++ {
				delete(s.eventCache, toDelete[i])
			}
		}
	}
}

func (s *Calendar) Stop() {

	if s.ticker == nil {
		return
	}
	log.Println("stopping ticker")
	s.ticker.Stop()
}

func (s *Calendar) getEvents(ctx context.Context, token *oauth2.Token, start time.Time, d time.Duration) error {

	client := s.oauth2Config.Client(ctx, token)

	end := start.Add(d)
	urlStr := fmt.Sprintf("https://outlook.office.com/api/beta/me/ReminderView(StartDateTime='%s',EndDateTime='%s')",
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	resp, err := client.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, string(b))
	}

	var events Events
	err = json.Unmarshal(b, &events)
	if err != nil {
		return err
	}

	log.Printf("refreshing events for the next %s - found %d events\n", d.String(), len(events.Events))

	for i := 0; i < len(events.Events); i++ {
		s.eventCache[events.Events[i].ID] = &events.Events[i]
		log.Printf("---> %s [reminder @ %s]\n", events.Events[i].Subject, events.Events[i].ReminderFireTime.TimeLocal(s.tz).Format(time.Kitchen))
	}

	return nil
}
