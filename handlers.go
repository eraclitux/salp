// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

package main

import (
	"encoding/json"
	"net/http"

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
			Type: "ghwebhook",
			Data: &pushData,
		}
	}
}
