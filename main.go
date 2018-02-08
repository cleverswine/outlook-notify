package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"./calendar"
	"./notify"
	"./store"
	oidc "github.com/coreos/go-oidc"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	appHost                  = flag.String("http", "localhost:5500", "host:port to use for this application's http server")
	clientID                 = flag.String("client", "", "A client that is registered in MS AS with appropraite permissions")
	clientSecret             = flag.String("secret", "", "The client secret")
	msTenant                 = flag.String("tenant", "common", "The MS directory to use for login")
	icon                     = flag.String("icon", "/usr/share/icons/Mint-X-Dark/status/24/stock_appointment-reminder.png", "Icon to use for notifications")
	timeFormat               = flag.String("timeformat", time.Kitchen, "Display format for reminder times")
	tickerIntervalSecs       = flag.Int("ticker", 30, "Frequency of reminder checks in seconds")
	refreshIntervalMinutes   = flag.Int("refresh", 15, "Frequency of refreshing event data from the Graph API in minutes")
	lookAheadIntervalMinutes = flag.Int("lookahead", 60, "Minutes of lookahead data to get from calendar")
	localTZ                  = flag.String("tz", "America/Los_Angeles", "Local time zone")
	debug                    = flag.Bool("debug", false, "enable verbose logging")
)

func main() {

	flag.Parse()
	if *clientID == "" || *clientSecret == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appURL := "http://" + *appHost
	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0", *msTenant)

	oauth2Config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: authURL + "/authorize", TokenURL: authURL + "/token"},
		RedirectURL:  appURL + "/callback",
		Scopes:       []string{oidc.ScopeOpenID, "Calendars.Read", "User.Read", "offline_access"},
	}

	// Watch for events
	ch := make(chan *calendar.Event, 10)
	defer close(ch)
	go func() {
		var notifier notify.Notifier
		notifier = notify.NewNotifySend(*icon)
		for evt := range ch {
			fmt.Println(evt.Subject)
			notifier.Send(evt.Subject, evt.String(*localTZ, *timeFormat))
		}
	}()

	// Start up a new calendar process any time we get a new token
	tokenCh := make(chan *oauth2.Token, 10)
	defer close(tokenCh)
	go func() {
		var cal *calendar.Calendar
		for token := range tokenCh {
			log.Println("got a token")
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
	tokenStore := store.NewTokenStore("token.json")
	token := tokenStore.Get()
	if token != nil {
		tokenCh <- token
	}

	// everything below here is for acuiring a new auth token
	var state string

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Nothing here"))
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if *debug {
			log.Println("redirecting from: " + r.URL.RequestURI())
		}
		stateUUID, err := uuid.NewV4()
		if err != nil {
			http.Error(w, "Failed to get state UUID: "+err.Error(), http.StatusInternalServerError)
			return
		}
		state = stateUUID.String()
		http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if *debug {
			log.Println("callback: " + r.URL.RequestURI())
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
		token, err := oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tokenStore.Save(token)
		tokenCh <- token

		w.Write([]byte("OK"))
	})

	log.Printf("listening on " + *appHost)
	log.Fatal(http.ListenAndServe(*appHost, nil))
}
