package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"./calendar"
	"./notify"
	"./server"
	"./store"
	"github.com/TV4/graceful"
	oidc "github.com/coreos/go-oidc"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	port                     = flag.String("port", "5500", "host:port to use for this application's http server")
	clientID                 = flag.String("client", "", "A client that is registered in MS AS with appropraite permissions")
	clientSecret             = flag.String("secret", "", "The client secret")
	msTenant                 = flag.String("tenant", "common", "The MS directory to use for login")
	icon                     = flag.String("icon", "/usr/share/icons/gnome/32x32/status/appointment-soon.png", "Icon to use for notifications")
	timeFormat               = flag.String("timeformat", time.Kitchen, "Display format for reminder times")
	tickerIntervalSecs       = flag.Int("ticker", 30, "Frequency of reminder checks in seconds")
	refreshIntervalMinutes   = flag.Int("refresh", 15, "Frequency of refreshing event data from the Graph API in minutes")
	lookAheadIntervalMinutes = flag.Int("lookahead", 60, "Minutes of lookahead data to get from calendar")
	localTZ                  = flag.String("tz", "America/Los_Angeles", "Local time zone")
	debug                    = flag.Bool("debug", false, "enable verbose logging")
	dryRun                   = flag.Bool("dry-run", false, "show a test notification")
	help                     = flag.Bool("help", false, "show help")
)

func main() {

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *dryRun {
		notify.NewNotifySend(*icon).Send("Meet with the Bobs", "2:00P - 2:30P  Room A\nhttps://outlook.office365.com/owa/?itemid=somereallylongidentifierthatlookslikeahash&path=/calendar/item")
		os.Exit(0)
	}

	validateArgs()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// oauth2 configuration for interacting with the MS Graph API
	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0", url.PathEscape(*msTenant))
	oauth2Config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: authURL + "/authorize", TokenURL: authURL + "/token"},
		RedirectURL:  "http://localhost:" + *port + "/callback",
		Scopes:       []string{oidc.ScopeOpenID, "Calendars.Read", "User.Read", "offline_access"},
	}

	// Watch for calendar events
	ch := make(chan *calendar.Event, 10)
	defer close(ch)
	go func() {
		var notifier notify.Notifier
		notifier = notify.NewNotifySend(*icon)
		for evt := range ch {
			log.Printf("[notify] %s / %s (reminder @ %s)", evt.Subject, evt.String(*localTZ, *timeFormat), evt.ReminderFireTime.TimeLocal(*localTZ).Format(*timeFormat))
			notifier.Send(evt.Subject, evt.String(*localTZ, *timeFormat))
		}
	}()

	tokenStore := store.NewTokenStore("token.json")

	// Start up a new calendar process any time we get a new token
	tokenCh := make(chan *oauth2.Token, 10)
	defer close(tokenCh)
	go func() {
		var cal *calendar.Calendar
		for token := range tokenCh {
			log.Println("got a token")
			tokenStore.Save(token)
			if cal != nil {
				cal.Stop()
			}
			cal = calendar.New(token, oauth2Config, *localTZ, ch)
			go cal.Start(ctx,
				time.Second*time.Duration(*tickerIntervalSecs),
				time.Minute*time.Duration(*refreshIntervalMinutes),
				time.Minute*time.Duration(*lookAheadIntervalMinutes))
		}
	}()

	// if we have a cached token, use it
	token := tokenStore.Get()
	if token != nil {
		tokenCh <- token
	}

	// start up an http server for getting auth tokens
	server := server.NewServer(oauth2Config, tokenCh, nil)
	graceful.LogListenAndServe(&http.Server{
		Addr:    ":" + *port,
		Handler: server,
	})
}

func validateArgs() {

	if *clientID == "" || *clientSecret == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *tickerIntervalSecs < 30 {
		log.Fatalln("ticker should be >= 30 seconds")
	}
	if *refreshIntervalMinutes < 5 {
		log.Fatalln("refresh should be >= 5 minutes")
	}
	if *lookAheadIntervalMinutes < 5 || *lookAheadIntervalMinutes > 1440 {
		log.Fatalln("refresh should be >= 5 minutes and <= 1440 minutes")
	}
	if _, err := os.Stat(*icon); err != nil {
		if err == os.ErrNotExist {
			log.Fatalf("Icon file '%s' does not exist\n", *icon)
		} else {
			log.Fatalf("Could not stat icon file '%s'\n", *icon)
		}
	}
}
