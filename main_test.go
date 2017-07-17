package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
)

func TestVoteHandler(t *testing.T) {

	s := mapStore{
		db: make(map[string]lunch),
	}

	l := lunch{Open: true}
	s.Store([]byte("lunch_1"), l)

	ls := lunchServer{
		s: &s,
	}

	req, err := http.NewRequest("POST", "/vote", strings.NewReader("payload=%7B%22actions%22%3A%5B%7B%22name%22%3A%22lunch%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%7D%5D%2C%22callback_id%22%3A%22lunch_1%22%2C%22team%22%3A%7B%22id%22%3A%22T03NPFSEK%22%2C%22domain%22%3A%22tangent-sap%22%7D%2C%22channel%22%3A%7B%22id%22%3A%22C03NPFSEV%22%2C%22name%22%3A%22general%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U03NPFSEP%22%2C%22name%22%3A%22tom%22%7D%2C%22action_ts%22%3A%221499622670.854364%22%2C%22message_ts%22%3A%221499622351.295776%22%2C%22attachment_id%22%3A%221%22%2C%22token%22%3A%22gorIxX00vvzuqHT7vye1ng63%22%2C%22is_app_unfurl%22%3Afalse%2C%22original_message%22%3A%7B%22text%22%3A%22What%27s+for+lunch%3F%22%2C%22username%22%3A%22slacky%22%2C%22bot_id%22%3A%22B63LB2N8N%22%2C%22mrkdwn%22%3Atrue%2C%22attachments%22%3A%5B%7B%22callback_id%22%3A%22lunch_1%22%2C%22text%22%3A%22Vote+below%3A%22%2C%22id%22%3A1%2C%22actions%22%3A%5B%7B%22id%22%3A%221%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Pret%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%2C%22style%22%3A%22%22%7D%2C%7B%22id%22%3A%222%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Leon%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%2C%22style%22%3A%22%22%7D%5D%7D%5D%2C%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22ts%22%3A%221499622351.295776%22%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT03NPFSEK%5C%2F210845919430%5C%2FXBJ9SILMS8tE21jH7cHBp2C0%22%7D"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("accept", "application/json,*/*")
	req.Header.Set("content-length", "1451")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	h := http.HandlerFunc(ls.voteHandler)
	h.ServeHTTP(w, req)

	l, err = s.Find([]byte("lunch_1"))
	if err != nil {
		t.Fatal(err)
	}

	if len(l.Votes) != 1 {
		t.Errorf("Unexpected number of votes. Expected %d got %d", 1, len(l.Votes))
	}
	if len(l.Votes) > 0 && l.Votes[0].Value != "leon" {
		t.Errorf("Vote should be been for leon")
	}
	if len(l.Votes) > 0 && l.Votes[0].UserName != "tom" {
		t.Errorf("Vote should be been made by tom")
	}
	if len(l.Votes) > 0 && l.Votes[0].UserID != "U03NPFSEP" {
		t.Errorf("Vote should be been made by ID U03NPFSEP")
	}
}

func TestUserCanOnlyVoteOnce(t *testing.T) {
	s := mapStore{
		db: make(map[string]lunch),
	}
	l := lunch{Open: true}
	s.Store([]byte("lunch_1"), l)

	ls := lunchServer{
		s: &s,
	}

	req, err := http.NewRequest("POST", "/vote", strings.NewReader("payload=%7B%22actions%22%3A%5B%7B%22name%22%3A%22lunch%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%7D%5D%2C%22callback_id%22%3A%22lunch_1%22%2C%22team%22%3A%7B%22id%22%3A%22T03NPFSEK%22%2C%22domain%22%3A%22tangent-sap%22%7D%2C%22channel%22%3A%7B%22id%22%3A%22C03NPFSEV%22%2C%22name%22%3A%22general%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U03NPFSEP%22%2C%22name%22%3A%22tom%22%7D%2C%22action_ts%22%3A%221499622670.854364%22%2C%22message_ts%22%3A%221499622351.295776%22%2C%22attachment_id%22%3A%221%22%2C%22token%22%3A%22gorIxX00vvzuqHT7vye1ng63%22%2C%22is_app_unfurl%22%3Afalse%2C%22original_message%22%3A%7B%22text%22%3A%22What%27s+for+lunch%3F%22%2C%22username%22%3A%22slacky%22%2C%22bot_id%22%3A%22B63LB2N8N%22%2C%22mrkdwn%22%3Atrue%2C%22attachments%22%3A%5B%7B%22callback_id%22%3A%22lunch_1%22%2C%22text%22%3A%22Vote+below%3A%22%2C%22id%22%3A1%2C%22actions%22%3A%5B%7B%22id%22%3A%221%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Pret%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%2C%22style%22%3A%22%22%7D%2C%7B%22id%22%3A%222%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Leon%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%2C%22style%22%3A%22%22%7D%5D%7D%5D%2C%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22ts%22%3A%221499622351.295776%22%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT03NPFSEK%5C%2F210845919430%5C%2FXBJ9SILMS8tE21jH7cHBp2C0%22%7D"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("accept", "application/json,*/*")
	req.Header.Set("content-length", "1451")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	h := http.HandlerFunc(ls.voteHandler)
	h.ServeHTTP(w, req)

	req2, err := http.NewRequest("POST", "/vote", strings.NewReader("payload=%7B%22actions%22%3A%5B%7B%22name%22%3A%22lunch%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%7D%5D%2C%22callback_id%22%3A%22lunch_1%22%2C%22team%22%3A%7B%22id%22%3A%22T03NPFSEK%22%2C%22domain%22%3A%22tangent-sap%22%7D%2C%22channel%22%3A%7B%22id%22%3A%22C03NPFSEV%22%2C%22name%22%3A%22general%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U03NPFSEP%22%2C%22name%22%3A%22tom%22%7D%2C%22action_ts%22%3A%221499622670.854364%22%2C%22message_ts%22%3A%221499622351.295776%22%2C%22attachment_id%22%3A%221%22%2C%22token%22%3A%22gorIxX00vvzuqHT7vye1ng63%22%2C%22is_app_unfurl%22%3Afalse%2C%22original_message%22%3A%7B%22text%22%3A%22What%27s+for+lunch%3F%22%2C%22username%22%3A%22slacky%22%2C%22bot_id%22%3A%22B63LB2N8N%22%2C%22mrkdwn%22%3Atrue%2C%22attachments%22%3A%5B%7B%22callback_id%22%3A%22lunch_1%22%2C%22text%22%3A%22Vote+below%3A%22%2C%22id%22%3A1%2C%22actions%22%3A%5B%7B%22id%22%3A%221%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Pret%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%2C%22style%22%3A%22%22%7D%2C%7B%22id%22%3A%222%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Leon%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%2C%22style%22%3A%22%22%7D%5D%7D%5D%2C%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22ts%22%3A%221499622351.295776%22%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT03NPFSEK%5C%2F210845919430%5C%2FXBJ9SILMS8tE21jH7cHBp2C0%22%7D"))
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("accept", "application/json,*/*")
	req2.Header.Set("content-length", "1451")
	req2.Header.Set("content-type", "application/x-www-form-urlencoded")

	h.ServeHTTP(w, req2)

	l, err = s.Find([]byte("lunch_1"))
	if err != nil {
		t.Fatal(err)
	}

	if len(l.Votes) != 1 {
		t.Errorf("Unexpected number of votes. Expected %d got %d", 1, len(l.Votes))
	}
}

func TestCannotVoteOnClosedVote(t *testing.T) {
	s := mapStore{
		db: make(map[string]lunch),
	}

	luh := lunch{}

	fmt.Printf("%v", luh.Open)

	s.Store([]byte("lunch_1"), luh)

	ls := lunchServer{
		s: &s,
	}

	req, err := http.NewRequest("POST", "/vote", strings.NewReader("payload=%7B%22actions%22%3A%5B%7B%22name%22%3A%22lunch%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%7D%5D%2C%22callback_id%22%3A%22lunch_1%22%2C%22team%22%3A%7B%22id%22%3A%22T03NPFSEK%22%2C%22domain%22%3A%22tangent-sap%22%7D%2C%22channel%22%3A%7B%22id%22%3A%22C03NPFSEV%22%2C%22name%22%3A%22general%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U03NPFSEP%22%2C%22name%22%3A%22tom%22%7D%2C%22action_ts%22%3A%221499622670.854364%22%2C%22message_ts%22%3A%221499622351.295776%22%2C%22attachment_id%22%3A%221%22%2C%22token%22%3A%22gorIxX00vvzuqHT7vye1ng63%22%2C%22is_app_unfurl%22%3Afalse%2C%22original_message%22%3A%7B%22text%22%3A%22What%27s+for+lunch%3F%22%2C%22username%22%3A%22slacky%22%2C%22bot_id%22%3A%22B63LB2N8N%22%2C%22mrkdwn%22%3Atrue%2C%22attachments%22%3A%5B%7B%22callback_id%22%3A%22lunch_1%22%2C%22text%22%3A%22Vote+below%3A%22%2C%22id%22%3A1%2C%22actions%22%3A%5B%7B%22id%22%3A%221%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Pret%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%2C%22style%22%3A%22%22%7D%2C%7B%22id%22%3A%222%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Leon%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%2C%22style%22%3A%22%22%7D%5D%7D%5D%2C%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22ts%22%3A%221499622351.295776%22%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT03NPFSEK%5C%2F210845919430%5C%2FXBJ9SILMS8tE21jH7cHBp2C0%22%7D"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("accept", "application/json,*/*")
	req.Header.Set("content-length", "1451")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	h := http.HandlerFunc(ls.voteHandler)

	h.ServeHTTP(w, req)

	l, err := s.Find([]byte("lunch_1"))
	if err != nil {
		t.Fatal(err)
	}

	if len(l.Votes) > 0 {
		t.Errorf("Closed vote: Should not have been possible to updated number of votes")
	}
}

func TestCloseHandlerClosesVote(t *testing.T) {
	s := mapStore{
		db: make(map[string]lunch),
	}

	luh := lunch{Open: true}

	s.Store([]byte("lunch_1"), luh)

	ls := lunchServer{
		s: &s,
	}

	req, err := http.NewRequest("POST", "/close-vote/lunch_1", strings.NewReader("payload=%7B%22actions%22%3A%5B%7B%22name%22%3A%22lunch%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%7D%5D%2C%22callback_id%22%3A%22lunch_1%22%2C%22team%22%3A%7B%22id%22%3A%22T03NPFSEK%22%2C%22domain%22%3A%22tangent-sap%22%7D%2C%22channel%22%3A%7B%22id%22%3A%22C03NPFSEV%22%2C%22name%22%3A%22general%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U03NPFSEP%22%2C%22name%22%3A%22tom%22%7D%2C%22action_ts%22%3A%221499622670.854364%22%2C%22message_ts%22%3A%221499622351.295776%22%2C%22attachment_id%22%3A%221%22%2C%22token%22%3A%22gorIxX00vvzuqHT7vye1ng63%22%2C%22is_app_unfurl%22%3Afalse%2C%22original_message%22%3A%7B%22text%22%3A%22What%27s+for+lunch%3F%22%2C%22username%22%3A%22slacky%22%2C%22bot_id%22%3A%22B63LB2N8N%22%2C%22mrkdwn%22%3Atrue%2C%22attachments%22%3A%5B%7B%22callback_id%22%3A%22lunch_1%22%2C%22text%22%3A%22Vote+below%3A%22%2C%22id%22%3A1%2C%22actions%22%3A%5B%7B%22id%22%3A%221%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Pret%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22pret%22%2C%22style%22%3A%22%22%7D%2C%7B%22id%22%3A%222%22%2C%22name%22%3A%22lunch%22%2C%22text%22%3A%22Leon%22%2C%22type%22%3A%22button%22%2C%22value%22%3A%22leon%22%2C%22style%22%3A%22%22%7D%5D%7D%5D%2C%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22ts%22%3A%221499622351.295776%22%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT03NPFSEK%5C%2F210845919430%5C%2FXBJ9SILMS8tE21jH7cHBp2C0%22%7D"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("accept", "application/json,*/*")
	req.Header.Set("content-length", "1451")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/close-vote/{key}", ls.closeVotingHandler)

	r.ServeHTTP(w, req)

	l, err := s.Find([]byte("lunch_1"))
	if err != nil {
		t.Fatal(err)
	}

	if l.Open == true {
		t.Errorf("Closed vote handler did not close vote")
	}

}

func TestIDGen(t *testing.T) {
	k := getCurrentKey()
	var validKey = regexp.MustCompile(`^lunch_[0-9]{1,2}$`)

	if !validKey.MatchString(k) {
		t.Errorf("Tested key did not match extepcted format: %s", k)
	}

}

// mapStore is a simplistic implmentation of the store interface
type mapStore struct {
	mu sync.RWMutex
	db map[string]lunch
}

func (s *mapStore) Find(key []byte) (lunch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.db[string(key)]
	if !ok {
		return lunch{}, fmt.Errorf("Could not find lunch for key %s", string(key))
	}

	return v, nil
}

func (s *mapStore) Store(k []byte, l lunch) (lunch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l.ID = k
	s.db[string(k)] = l

	return l, nil
}

func (s *mapStore) List() []lunch {
	var lunches []lunch
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.db {
		lunches = append(lunches, v)
	}

	return lunches
}
