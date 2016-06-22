// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/eraclitux/stracer"
	"github.com/nlopes/slack"
)

type Author struct {
	Name, Username string
}

type Commit struct {
	Message string
	Url     string
	Author  Author
}

type GHPushEvent struct {
	Ref        string
	Compare    string
	Commits    []Commit
	Repository map[string]interface{}
}

// FIXME rename and make webhook clear
type MessageEvent struct {
	Message string
	Type    string
}

type NewRelicEvent struct {
	Date             time.Time `json:"created_at"`
	ShortDescription string    `json:"short_description"`
	Message          string
	Severity         string
}

// GHWebhooksHandlerFunc deals with webhook events received from GitHub.
func GHWebhooksHandlerFunc(c chan<- slack.RTMEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stracer.Traceln(r.Header.Get("X-Hub-Signature"))
		event := r.Header.Get("X-GitHub-Event")
		// only push event supported for now
		if event != "push" {
			http.Error(w, `only "push" event supported`, http.StatusBadRequest)
			return
		}
		var pushData GHPushEvent
		err := json.NewDecoder(r.Body).Decode(&pushData)
		if err != nil {
			ErrorLogger.Println("decoding json body:", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		stracer.PrettyStruct("GHPushEvent", pushData)
		// send to RTM channel...
		c <- slack.RTMEvent{
			//Type: "ghwebhook", FIXME unused?
			Data: &pushData,
		}
	}
}

// GenericMessageHandler receives a generic JSON message
// in the form:
//	{
//		"message": string,
//		"type": string
//	}
// that echoes back to Slack.
func GenericMessageHandler(c chan<- slack.RTMEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var message MessageEvent
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			ErrorLogger.Println("decoding json body:", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		// send to RTM channel...
		c <- slack.RTMEvent{
			Data: &message,
		}
	}
}

func NewRelicHandler(c chan<- slack.RTMEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var message NewRelicEvent
		reader := strings.NewReader(r.PostFormValue("alert"))
		err := json.NewDecoder(reader).Decode(&message)
		stracer.Traceln(message)
		if err != nil {
			ErrorLogger.Println("decoding json body:", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		// send to RTM channel...
		c <- slack.RTMEvent{
			Data: &message,
		}
	}
}

// StatusHandlerFunc is a read only endpoint
// that returns aggregated data that Salp
// raceives from different sources (NewRelic, GitHub etc.)
func StatusHandlerFunc(w http.ResponseWriter, r *http.Request) {
	generalStatus.mu.RLock()
	defer generalStatus.mu.RUnlock()
	err := json.NewEncoder(w).Encode(generalStatus)
	if err != nil {
		ErrorLogger.Println("encoding json body:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// MustAuth is a decorator that implements a simple token
// authentication.
func MustAuth(authToken string, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stracer.Traceln(r.Header.Get("Salp-auth"))
		if r.Header.Get("Salp-Auth") != authToken {
			time.Sleep(time.Duration(rand.Intn(100)+100) * time.Millisecond)
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			stracer.Traceln("401")
			return
		}
		fn(w, r)
	}
}
