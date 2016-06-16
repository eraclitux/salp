// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eraclitux/stracer"
	"github.com/nlopes/slack"
)

func AmIMentioned(msg, myID string) bool {
	return strings.Contains(msg, myID)
}

func IsDirectMessage(channel string) bool {
	if strings.IndexRune(channel, 'D') == 0 {
		return true
	}
	return false
}

// getSecInfo retrieves sec info from istheinternetonfire.com
// and formats them.
func getSecInfo() string {
	errString := "unable to get security info\nhttps://istheinternetonfire.com"
	resp, err := http.Get("https://istheinternetonfire.com/status.json")
	if err != nil {
		stracer.Traceln("getSecInfo:", err)
		return errString
	}
	defer resp.Body.Close()
	data := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		stracer.Traceln("getSecInfo:", err)
		return errString
	}
	var answers string
	if issues, ok := data["issues"].([]interface{}); ok {
		for _, issue := range issues {
			i := map[string]interface{}{}
			if i, ok = issue.(map[string]interface{}); !ok {
				return errString
			}
			answ := ""
			answ, _ = i["txt"].(string)
			answers += fmt.Sprintf("%s,", answ)
		}
	} else {
		stracer.Traceln("getSecInfo: assertion error")
		stracer.PrettyStruct("data", data)
		return errString
	}
	return fmt.Sprintf(
		"%s %s (%s)\n%s",
		answers,
		data["status"],
		data["date"],
		"https://istheinternetonfire.com/",
	)
}

// FIXME only accept rtm.IncomingEvents chan as argument
func scheduleReminder(ev *slack.MessageEvent, rtm *slack.RTM) string {
	var toIndex, inIndex int
	var thingToDo string
	var minutes int
	errorMesage := "unable to decode reminder :white_frowning_face:\n" +
		"please use the form " +
		RemindMeFormat
	terms := strings.Fields(ev.Text)
	for i, term := range terms {
		switch term {
		case "to":
			// consider only the first
			if toIndex == 0 {
				toIndex = i
			}
		case "in":
			inIndex = i
		}
	}
	if toIndex+1 < inIndex {
		for _, term := range terms[toIndex+1 : inIndex] {
			thingToDo += term + " "
		}
	} else {
		return errorMesage
	}
	if inIndex+1 < len(terms) {
		var err error
		minutes, err = strconv.Atoi(terms[inIndex+1])
		if err != nil {
			return errorMesage
		}
	} else {
		return errorMesage
	}
	text := fmt.Sprintf("<@%s> remember to %s:robot_face:", ev.User, thingToDo)
	// FIXME is rtm concurrency-safe?
	// FIXME do not encole rtm, but on c := rtm.IncomingEvents
	// and send a signal to main cicle
	go func() {
		time.Sleep(time.Duration(minutes) * time.Minute)
		rtm.SendMessage(
			rtm.NewOutgoingMessage(text, ev.Channel),
		)
	}()
	return fmt.Sprintf("ok, I'll remind you in %d minutes :robot_face:", minutes)
}

// randomCharset contains the characters that can make up a randomString().
const randomCharset = "01234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"

// randomString returns a string of random characters (taken from
// randomCharset) of the specified length.
// Taken from syncthing.
func randomString(l int) string {
	bs := make([]byte, l)
	for i := range bs {
		bs[i] = randomCharset[rand.Intn(len(randomCharset))]
	}
	return string(bs)
}
