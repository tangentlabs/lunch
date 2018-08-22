package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
)

type lunch struct {
	ID       []byte        `json:"id"`
	Options  []lunchOption `json:"options"`
	Votes    []vote        `json:"votes"`
	Question string        `json:"question"`
	Open     bool          `json:"open"`
}

type lunchOption struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

type vote struct {
	Value    string `json:"value"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

func main() {

	db, err := bolt.Open("lunch.db", 0600, nil)
	if err != nil {
		log.Fatalf("Could not open DB: %s", err)
	}
	defer db.Close()

	tx, err := db.Begin(true)
	if err != nil {
		log.Fatalf("Could not see bucket: %s", err)
	}
	tx.CreateBucketIfNotExists([]byte("Lunches"))
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Something went very wrong: %s", err)
	}

	bs := boltStore{db: db}

	api := slack.New(os.Getenv("SLACKKEY"))
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)

	c := comsClient{slack: api}

	ls := lunchServer{s: bs, c: c}

	r := mux.NewRouter()

	r.HandleFunc("/", ls.indexHandler)
	r.HandleFunc("/view-lunch/{key}", ls.viewHandler)
	r.HandleFunc("/submit-lunch", ls.submitLunchHandler)
	r.HandleFunc("/vote", ls.voteHandler)
	r.HandleFunc("/close-vote/{key}", ls.closeVotingHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8765",
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

type lunchServer struct {
	s store
	c comsClient
}

type store interface {
	Find([]byte) (lunch, error)
	Store([]byte, lunch) (lunch, error)
	List() []lunch
}

type comsClient struct {
	slack *slack.Client
}

func (c *comsClient) PostMessage(dest string, message string, l lunch) error {
	params := lunchToSlackMsg(l)
	channelID, timestamp, err := c.slack.PostMessage(dest, message, params)
	if err != nil {
		return fmt.Errorf("Could not send to slack %s:%s %s: ", channelID, timestamp, err.Error())
	}
	return nil
}

func (s *lunchServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	id := getCurrentKey()
	if l, err := s.s.Find([]byte(id)); err == nil && l.ID != nil {
		// Lunch already exists
		http.Redirect(w, r, "/view-lunch/"+string(l.ID), http.StatusFound)
		return
	}

	t, _ := template.ParseFiles("static/index.html")
	t.Execute(w, nil)
	return
}

func (s *lunchServer) navigationHandler(w http.ResponseWriter, r *http.Request) {

}

func (s *lunchServer) viewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	l, err := s.s.Find([]byte(key))
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not find lunch %s: ", err.Error()), http.StatusInternalServerError)
		return
	}

	countedVotes := make(map[string]int)
	for _, v := range l.Votes {
		countedVotes[v.Value]++
	}

	v := struct {
		ID       string
		Question string
		Votes    map[string]int
		Open     bool
	}{
		string(l.ID),
		l.Question,
		countedVotes,
		l.Open,
	}

	t, _ := template.ParseFiles("static/view.html")
	t.Execute(w, v)
	return
}

func (s *lunchServer) submitLunchHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// does form have message and at least 2 choices

	id := getCurrentKey()
	if l, err := s.s.Find([]byte(id)); err == nil && l.ID != nil {
		http.Error(w, fmt.Sprintf("Lunch vote already exists for ID: %s", id), http.StatusInternalServerError)
		return
	}

	l := lunch{}
	l.ID = []byte(id)
	l.Question = r.PostForm.Get("msg")

	opts := []string{"1", "2", "3", "4", "5"}

	for _, o := range opts {
		v := r.PostForm.Get(strings.Join([]string{"option", o}, ""))
		if len(v) > 0 {
			lunchOpt := lunchOption{
				Text:  v,
				Value: v,
			}

			l.Options = append(l.Options, lunchOpt)
		}
	}
	l.Open = true

	l, err := s.s.Store(l.ID, l)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not save lunch %s: ", err.Error()), http.StatusInternalServerError)
		return
	}

	err = s.c.PostMessage("#general", "What's for lunch?", l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/view-lunch/"+string(l.ID), http.StatusFound)
}

func (s *lunchServer) voteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	p := r.PostForm.Get("payload")

	callbk := slack.AttachmentActionCallback{}
	json.Unmarshal([]byte(p), &callbk)

	l, err := s.s.Find([]byte(callbk.CallbackID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !l.Open {
		http.Error(w, "Vote is closed", http.StatusInternalServerError)
		return
	}

	// There is only ever one action. todo(tom): Add link to docs
	a := callbk.Actions[0]
	u := callbk.User

	// Does vote exist for this user
	for _, prevVote := range l.Votes {
		if prevVote.UserID == u.ID {
			http.Error(w, "User has already voted", http.StatusInternalServerError)
			return
		}
	}

	v := vote{
		Value:    a.Value,
		UserID:   u.ID,
		UserName: u.Name,
	}
	l.Votes = append(l.Votes, v)

	_, err = s.s.Store(l.ID, l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *lunchServer) closeVotingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	l, err := s.s.Find([]byte(key))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	l.Open = false

	_, err = s.s.Store([]byte(key), l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view-lunch/"+string(l.ID), http.StatusFound)
}

func lunchToSlackMsg(l lunch) slack.PostMessageParameters {
	params := slack.PostMessageParameters{}
	attachment := slack.Attachment{
		Text:       l.Question,
		CallbackID: string(l.ID),
		Actions:    []slack.AttachmentAction{},
	}

	for _, o := range l.Options {
		attachment.Actions = append(
			attachment.Actions,
			slack.AttachmentAction{
				Name:  "lunch", // What is name?
				Text:  o.Text,
				Value: o.Value,
				Type:  "button",
			})
	}

	params.Attachments = []slack.Attachment{attachment}

	return params
}

type boltStore struct {
	db *bolt.DB
}

func (s boltStore) Find(key []byte) (lunch, error) {
	l := lunch{}

	err := s.db.View(func(tx *bolt.Tx) error {
		lunches := tx.Bucket([]byte("Lunches"))

		lBytes := lunches.Get(key)

		err := json.Unmarshal(lBytes, &l)
		if err != nil {
			return fmt.Errorf("Could not unmarshal DB responses: %s", err)
		}

		return nil
	})

	// If there was no value for key bucket.Get() will return nil,
	// causing an unmarshal error - signifiying non existance

	return l, err
}

func (s boltStore) Store(k []byte, l lunch) (lunch, error) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		lunches, err := tx.CreateBucketIfNotExists([]byte("Lunches"))
		if err != nil {
			log.Fatalf("Could not open Lunches bucket: %s", err)
		}

		buf, err := json.Marshal(l)
		if err != nil {
			return fmt.Errorf("Could not store lunch: %s", err)
		}

		l.ID = k
		return lunches.Put(l.ID, buf)
	})

	return l, err
}

func (s boltStore) List() []lunch {
	lunchList := []lunch{}

	s.db.View(func(tx *bolt.Tx) error {
		lunches, err := tx.CreateBucketIfNotExists([]byte("Lunches"))
		if err != nil {
			log.Fatalf("Could not open Lunches bucket: %s", err)
		}

		lunches.ForEach(func(k, v []byte) error {
			l := lunch{}
			err := json.Unmarshal(v, &l)
			if err != nil {
				log.Fatal(err)
			}
			lunchList = append(lunchList, l)
			return nil
		})
		return nil
	})

	return lunchList
}

func getCurrentKey() string {
	_, weekNo := time.Now().ISOWeek()
	id := strings.Join([]string{"lunch_", strconv.Itoa(weekNo)}, "")
	return id
}
