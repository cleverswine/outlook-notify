package calendar

import (
	"log"
	"strings"
	"time"
)

type EventTime struct {
	DateTime string `json:"DateTime"`
	TimeZone string `json:"TimeZone"`
}

func (s EventTime) Time() time.Time {

	frontPart := strings.Split(string(s.DateTime), ".")[0]
	st, err := time.Parse("2006-01-02T15:04:05", frontPart)
	if err != nil {
		log.Fatal(err)
	}
	return st
}

func (s EventTime) TimeLocal(z string) time.Time {

	location, _ := time.LoadLocation(z)
	return s.Time().In(location)
}
