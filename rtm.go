// Copyright (c) 2016 Andrea Masi. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/eraclitux/stracer"
	"github.com/nlopes/slack"
)

const (
	UnknownAction = iota
	GreetAction
	InternetOnFireAction
	ReminderAction
	RemindMeFormat       = "`remind me to <thing to do> in <dd> minutes`"
	InternetOnFireFormat = "`is internet on fire`"
)

// ServeRTM deals with realtime messages from Slack.
// It is indended to tun in its own goroutine.
func ServeRTM(rtm *slack.RTM) {
	var slackInfo *slack.Info
	defer wg.Done()
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			stracer.Traceln("Event Received:")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				stracer.PrettyStruct("Infos", ev.Info)
				slackInfo = ev.Info
				stracer.Tracef("UserDetails: %+v\n", slackInfo.User)
				for _, c := range ev.Info.Channels {
					stracer.PrettyStruct("channel", c)
				}
				stracer.Traceln("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				stracer.PrettyStruct("Message", ev)
				ParseMessage(ev, rtm, slackInfo)

			case *slack.PresenceChangeEvent:
				//stracer.Tracef("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				stracer.Tracef("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				ErrorLogger.Println(ev.Error())

			case *slack.InvalidAuthEvent:
				ErrorLogger.Println("Invalid credentials")
				break Loop

			case *GHPushEvent:
				SendPushMessage(ev, rtm)

			case *MessageEvent:
				SendMessage(ev, rtm)

			default:
				stracer.PrettyStruct("unknown event:", ev)
			}
		}
	}
}

// SendPushMessage sends push event data to all channels
// of which Salp is member.
func SendPushMessage(pushData *GHPushEvent, rtm *slack.RTM) {
	channels, err := rtm.GetChannels(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	groups, err := rtm.GetGroups(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	var commitMessages, lastCommitUsername string
	for _, commit := range pushData.Commits {
		lastCommitUsername = commit.Author.Username
		commitMessages += commit.Message + "\n"
	}
	strings.Trim(commitMessages, "\n")
	text := fmt.Sprintf(
		"push on GitHub by `%s` on `%s`\n```%s```\ncompare: %s",
		lastCommitUsername,
		pushData.Ref,
		commitMessages,
		pushData.Compare,
	)
	// FIXME duplicated code
	for _, channel := range channels {
		stracer.PrettyStruct("channel", channel)
		if channel.IsMember {
			rtm.SendMessage(
				rtm.NewOutgoingMessage(text, channel.ID),
			)
		}
	}
	// FIXME duplicated code
	for _, group := range groups {
		stracer.PrettyStruct("group", group)
		rtm.SendMessage(
			rtm.NewOutgoingMessage(text, group.ID),
		)
	}
}

// SendMessage sends generic message received on /message
//to all channels of which Salp is member.
func SendMessage(message *MessageEvent, rtm *slack.RTM) {
	channels, err := rtm.GetChannels(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	groups, err := rtm.GetGroups(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	text := message.Message
	// FIXME duplicated code
	for _, channel := range channels {
		stracer.PrettyStruct("channel", channel)
		if channel.IsMember {
			rtm.SendMessage(
				rtm.NewOutgoingMessage(text, channel.ID),
			)
		}
	}
	// FIXME duplicated code
	for _, group := range groups {
		stracer.PrettyStruct("group", group)
		rtm.SendMessage(
			rtm.NewOutgoingMessage(text, group.ID),
		)
	}
}

func ParseMessage(ev *slack.MessageEvent, rtm *slack.RTM, slackInfo *slack.Info) {
	// discard my own message
	// that is received when connected
	myID := slackInfo.User.ID
	if ev.User == myID {
		return
	}
	if ev.SubType == "message_changed" {
		return
	}
	if AmIMentioned(ev.Text, myID) || IsDirectMessage(ev.Channel) {
		DecodeAndExecuteAction(ev, rtm)
	}
}

func DecodeAndExecuteAction(ev *slack.MessageEvent, rtm *slack.RTM) {
	var text string
	// FIXME create a normalization function
	ev.Text = strings.ToLower(ev.Text)
	actions := GuessActions(ev.Text)
	for _, action := range actions {
		switch action {
		case GreetAction:
			greeting := Greetings[rand.Intn(len(Greetings))]
			text += fmt.Sprintf("%s <@%s> :smile:\n", greeting, ev.User)
		case InternetOnFireAction:
			text += fmt.Sprintf("%s\n", getSecInfo())
		case ReminderAction:
			text += fmt.Sprintf(
				"%s\n",
				scheduleReminder(ev, rtm),
			)
		default:
		}
	}
	strings.Trim(text, "\n")
	if len(actions) == 0 {
		text = fmt.Sprintf(
			"I'm not that smart <@%s> :white_frowning_face:, you can ask me:\n%s\n%s",
			ev.User,
			InternetOnFireFormat,
			RemindMeFormat,
		)
	}
	rtm.SendMessage(
		rtm.NewOutgoingMessage(text, ev.Channel),
	)
}

func GuessActions(msg string) []int {
	actions := []int{}
	for _, word := range strings.Fields(msg) {
		for _, greeting := range Greetings {
			if word == greeting {
				actions = append(actions, GreetAction)
			}
		}
	}
	if strings.Contains(msg, "is internet on fire") {
		actions = append(actions, InternetOnFireAction)
	}

	if strings.Contains(msg, "remind me to") {
		actions = append(actions, ReminderAction)
	}
	stracer.Traceln(actions)
	return actions
}
