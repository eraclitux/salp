// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

// SALP - Slackbot Assistant for Lazy Programmers
package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/eraclitux/cfgp"
	"github.com/nlopes/slack"
)

type DaemonConf struct {
	Httpport      string
	Httpaddress   string
	SlackToken    string `cfgp:",Slack API token,"`
	NewRelicToken string `cfgp:"nr-token,NewRelic API token,"`
	NewRelicAppID string `cfgp:"nr-id,NewRelic app ID,"`
	AuthToken     string `cfgp:",token to authorize http requests,"`
	Version       bool   `cfgp:"v,show version and exit,"`
}

type status struct {
	Summary      string    `json:"summary"`
	Apdex        float64   `json:"apdex"`
	NRHealth     string    `json:"newrelic_health"`
	LastNRReport time.Time `json:"last_newrelic_reported"`
	mu           sync.RWMutex
}

func (s *status) updateNewRelicData(data NewRelicAppData) {
	s.LastNRReport = data.LastReported
	s.Apdex = data.Summary.Apdex
	s.NRHealth = data.Health
	s.Summary = s.Summarize()
}

func (s *status) Summarize() string {
	return "Health: " + s.NRHealth
}

// generalStatus stores aggregated info that Salp
// receives from different sources (NewRelic, GitHub etc.)
var generalStatus = status{
	Summary: "no data yet",
	Apdex:   0,
}

var wg sync.WaitGroup

// ErrorLogger is used to log error messages.
var ErrorLogger *log.Logger

// InfoLogger is used to log general info events.
var InfoLogger *log.Logger

// BUG(eraclitux): authenticate GH requests!
func main() {
	rand.Seed(time.Now().UnixNano())

	var conf DaemonConf
	conf = DaemonConf{
		Httpport:  "8080",
		AuthToken: randomString(8),
	}
	err := cfgp.Parse(&conf)
	if err != nil {
		log.Fatalln("parsing conf", err)
	}

	badExitCode := false
	SetupLoggers(os.Stdout, os.Stderr)

	if conf.NewRelicToken != "" && conf.NewRelicAppID != "" {
		go FetchNewRelic(conf.NewRelicToken, conf.NewRelicAppID)
	}

	api := slack.New(conf.SlackToken)
	rtm := api.NewRTM()
	if conf.SlackToken != "" {
		go rtm.ManageConnection()
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					// TODO try a last message sending to Slack via REST
					// user PostMessage
					ErrorLogger.Println(r)
				}
			}()
			ServeRTM(rtm)
		}()
	}

	http.HandleFunc("/status", StatusHandlerFunc)
	http.HandleFunc(
		"/gh-webhooks",
		GHWebhooksHandlerFunc(rtm.IncomingEvents),
	)
	http.HandleFunc(
		"/message",
		MustAuth(
			conf.AuthToken,
			GenericMessageHandler(rtm.IncomingEvents),
		),
	)
	http.HandleFunc(
		"/new-relic",
		NewRelicHandler(rtm.IncomingEvents),
	)
	addrString := fmt.Sprintf("%s:%s", conf.Httpaddress, conf.Httpport)
	InfoLogger.Println("start listening on:", addrString)
	if err := http.ListenAndServe(addrString, nil); err != nil {
		ErrorLogger.Println(err)
		badExitCode = true
	}

	wg.Wait()
	// FIXME send a message to RTM goroutine or deadlock
	if badExitCode {
		os.Exit(1)
	}
}

func SetupLoggers(i io.Writer, e io.Writer) {
	ErrorLogger = log.New(e, "[ERROR] ", log.Ldate|log.Ltime)
	InfoLogger = log.New(i, "[INFO] ", log.Ldate|log.Ltime)
}
