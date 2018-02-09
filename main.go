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
	port                     = flag.String("port", "5500", "Port to use for this application's http server")
	clientID                 = flag.String("client", "", "A client that is registered in MS AS with appropraite permissions")
	clientSecret             = flag.String("secret", "", "The client secret")
	msTenant                 = flag.String("tenant", "common", "The MS directory to use for login")
	notifierCmd              = flag.String("notifier", "zenity", "Application to use for notifications. options: zenity, notify-send")
	timeFormat               = flag.String("timeformat", time.Kitchen, "Display format for reminder times")
	tickerIntervalSecs       = flag.Int("ticker", 30, "Frequency of reminder checks in seconds")
	refreshIntervalMinutes   = flag.Int("refresh", 15, "Frequency of refreshing event data from the Graph API in minutes")
	lookAheadIntervalMinutes = flag.Int("lookahead", 60, "Minutes of lookahead data to get from calendar")
	localTZ                  = flag.String("tz", "America/Los_Angeles", "Local time zone")
	debug                    = flag.Bool("debug", false, "Enable verbose logging")
	dryRun                   = flag.Bool("dry-run", false, "Show a test notification")
	help                     = flag.Bool("help", false, "Show this help")
)

func main() {

	flag.Parse()
	validateArgs()

	notifier := selectNotifier()

	if *dryRun {
		notifier.Send("Meet with the Bobs", "2:00P - 2:30P  Room A")
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// oauth2 configuration for interacting with the MS Graph API
	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0", url.PathEscape(*msTenant))
	oauth2Config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: authURL + "/authorize", TokenURL: authURL + "/token"},
		RedirectURL:  "http://localhost:" + *port + "/callback",
		Scopes:       []string{oidc.ScopeOpenID, "Calendars.Read", "offline_access"},
	}

	// Watch for calendar events
	ch := make(chan *calendar.Event, 10)
	defer close(ch)
	go func() {
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
		cal := calendar.New(oauth2Config,
			time.Second*time.Duration(*tickerIntervalSecs),
			time.Minute*time.Duration(*refreshIntervalMinutes),
			time.Minute*time.Duration(*lookAheadIntervalMinutes),
			*localTZ, ch)

		for token := range tokenCh {
			log.Println("got a token")
			tokenStore.Save(token)
			cal.Stop()
			go cal.Start(ctx, token)
		}
	}()

	// if we have a cached token, use it
	token := tokenStore.Get()
	if token != nil {
		tokenCh <- token
	} else {
		notifier.Send("Authentication Required", "Please visit http://localhost:"+*port+" and log in to your MS account")
	}

	// start up an http server for getting auth tokens
	server := server.NewServer(oauth2Config, tokenCh, nil)
	graceful.LogListenAndServe(&http.Server{
		Addr:    ":" + *port,
		Handler: server,
	})
}

func validateArgs() {

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if !*dryRun && (*clientID == "" || *clientSecret == "") {
		log.Fatalln("clientID and clientSecret are required")
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
}

func selectNotifier() notify.Notifier {

	// pick a notifier service (TODO make sure the specified notifierCmd is actually installed)
	switch *notifierCmd {
	case "zenity":
		return notify.NewZenity()
	case "notify-send":
		// TODO make sure the icon exists
		return notify.NewNotifySend("/usr/share/icons/gnome/32x32/status/appointment-soon.png")
	default:
		log.Fatalln("Invalid notifer specified")
		return nil
	}
}
