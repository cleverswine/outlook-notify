package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	oidc "github.com/coreos/go-oidc"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	appHost                  = flag.String("http", "localhost:5500", "host:port to use for this application's http server")
	clientID                 = flag.String("client", "", "A client that is registered in MS AS with appropraite permissions")
	clientSecret             = flag.String("secret", "", "The client secret")
	icon                     = flag.String("icon", "/usr/share/icons/Mint-X-Dark/status/24/stock_appointment-reminder.png", "Icon to use for notifications")
	timeFormat               = flag.String("timeformat", time.Kitchen, "Display format for reminder times")
	tickerIntervalSecs       = flag.Int("ticker", 30, "Frequency of reminder checks in seconds")
	lookAheadIntervalMinutes = flag.Int("lookahead", 60, "Minutes of lookahead data to get from calendar")
	msTenant                 = flag.String("tenant", "common", "The MS directory to use for login")
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

	calendar := NewCalendar(time.Minute * time.Duration(*lookAheadIntervalMinutes))
	tokenStore := &TokenStore{FileName: "token.json"}
	token := tokenStore.Get()

	config := oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: authURL + "/authorize", TokenURL: authURL + "/token"},
		RedirectURL:  appURL + "/callback",
		Scopes:       []string{oidc.ScopeOpenID, "Calendars.Read", "User.Read", "offline_access"},
	}

	// cache upcoming events
	calendar.RefreshReminders(config.Client(ctx, token))
	// only notify about an empty token once
	notifiedAboutEmptyToken := false

	refreshTicker := time.NewTicker(time.Minute * time.Duration(*lookAheadIntervalMinutes))
	defer refreshTicker.Stop()
	go func() {
		for _ = range refreshTicker.C {
			if token == nil && !notifiedAboutEmptyToken {
				exec.Command("notify-send", "Calendar", "No token found. Please visit "+appURL+" to get one.")
				notifiedAboutEmptyToken = true
				continue
			}
			err := calendar.RefreshReminders(config.Client(ctx, token))
			if err != nil {
				exec.Command("notify-send", "Calendar", "Unable to get reminders: "+err.Error())
				continue
			}
			tokenStore.Save(token)
		}
	}()

	// look for events that need reminders to be shown
	ticker := time.NewTicker(time.Second * time.Duration(*tickerIntervalSecs))
	defer ticker.Stop()
	go func() {
		for _ = range ticker.C {
			calendar.SendReminders()
		}
	}()

	// everything below here is for acuiring the initial auth token
	var state string

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		stateUUID, err := uuid.NewV4()
		if err != nil {
			http.Error(w, "Failed to get state UUID: "+err.Error(), http.StatusInternalServerError)
			return
		}
		state = stateUUID.String()
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		token = oauth2Token
		tokenStore.Save(token)
		w.Write([]byte("OK"))
	})

	log.Printf("listening on " + *appHost)
	log.Fatal(http.ListenAndServe(*appHost, nil))
}
