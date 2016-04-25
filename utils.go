// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

func scheduleReminder(ev *slack.MessageEvent, rtm *slack.RTM) string {
	// FIXME are concurrency-safe?
	return "unable to decode reminder :-("
}
