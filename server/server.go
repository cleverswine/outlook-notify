package server

import (
	"log"
	"net/http"
	"os"
	"time"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/oauth2"
)

type Server struct {
	oauth2Config *oauth2.Config
	ch           chan *oauth2.Token
	logger       *log.Logger
	mux          *http.ServeMux
}

func NewServer(oauth2Config *oauth2.Config, ch chan *oauth2.Token, logger *log.Logger) *Server {

	s := &Server{
		mux:          http.NewServeMux(),
		oauth2Config: oauth2Config,
		ch:           ch,
		logger:       logger,
	}

	if s.logger == nil {
		s.logger = log.New(os.Stdout, "", 0)
	}

	s.mux.HandleFunc("/", s.index)
	s.mux.HandleFunc("/token", s.token)
	s.mux.HandleFunc("/callback", s.callback)

	return s
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {

	stateUUID, err := uuid.NewV4()
	if err != nil {
		http.Error(w, "Failed to get state UUID: "+err.Error(), http.StatusInternalServerError)
		return
	}
	state := stateUUID.String()
	http.SetCookie(w, &http.Cookie{Name: "state", Value: state, HttpOnly: true, Expires: time.Now().Add(time.Minute * 5)})

	w.Write([]byte("<html><head><title>Outlook Notifier</title></head><body><p><a href='/token'>Sign in</a> to your Microsoft account to start monitoring calendar events</p></body></html>"))
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {

	if cookie, err := r.Cookie("state"); err == nil {
		log.Println(cookie.Value)
		http.Redirect(w, r, s.oauth2Config.AuthCodeURL(cookie.Value), http.StatusFound)
	} else {
		http.Error(w, "no state cookie found", http.StatusBadRequest)
		return
	}
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {

	if cookie, err := r.Cookie("state"); err == nil {
		log.Println(cookie.Value)
		if cookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "no state cookie found", http.StatusBadRequest)
		return
	}

	token, err := s.oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.ch <- token

	// remove state cookie
	http.SetCookie(w, &http.Cookie{Name: "state", MaxAge: -1})

	w.Write([]byte("OK"))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	s.mux.ServeHTTP(w, r)
}
