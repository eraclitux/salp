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

type Connection struct {
	SlackInfo *slack.Info
	RTM       *slack.RTM
}

// ParseMessage decodes and executes messages from humans received
// in Slack channels and groups to which Salp belongs.
func (c Connection) ParseMessage(ev *slack.MessageEvent) {
	// discard my own message
	// that is received when connected
	myID := c.SlackInfo.User.ID
	if ev.User == myID {
		return
	}
	if ev.SubType == "message_changed" {
		return
	}
	if AmIMentioned(ev.Text, myID) || IsDirectMessage(ev.Channel) {
		c.decodeAndExecuteAction(ev)
	}
}

// decodeAndExecuteAction deals with Slack messages from humans.
func (c Connection) decodeAndExecuteAction(ev *slack.MessageEvent) {
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
			stracer.Traceln("scheduling reminder...")
			text += fmt.Sprintf(
				"%s\n",
				scheduleReminder(ev, c.RTM),
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
	c.sendSlackMessage(text, ev.Channel)
}

// SendMessage redirects generic messages received on /message
// http endpoint to all channels and groups of which Salp is member.
func (c Connection) SendMessage(message *MessageEvent) {
	text := message.Message
	c.toAllChannels(text)
	c.toAllGroups(text)
}

func (c Connection) SendNewRelicMessage(message *NewRelicEvent) {
	text := fmt.Sprintf(
		"New Relic event, severity: `%s` on `%s`\n```%s```",
		message.Severity,
		message.Date,
		message.Message,
	)
	c.toAllChannels(text)
	c.toAllGroups(text)
}

// SendPushMessage sends push event data to all channels
// and groups of which Salp is member.
// BUG(eraclitux): discard non push messages
func (c Connection) SendPushMessage(pushData *GHPushEvent) {
	var commitMessages, lastCommitUsername string
	commits := pushData.Commits
	commitsCount := len(commits)
	if commitsCount > 0 {
		lastCommitUsername = commits[commitsCount-1].Author.Username
	}
	for i := commitsCount - 1; i >= 0; i-- {
		commit := commits[i]
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
	c.toAllChannels(text)
	c.toAllGroups(text)
}

func (c Connection) toAllGroups(text string) {
	groups, err := c.RTM.GetGroups(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	for _, group := range groups {
		stracer.PrettyStruct("group", group)
		c.sendSlackMessage(text, group.ID)
	}
}

func (c Connection) toAllChannels(text string) {
	channels, err := c.RTM.GetChannels(true)
	if err != nil {
		ErrorLogger.Println(err)
		return
	}
	for _, channel := range channels {
		stracer.PrettyStruct("channel", channel)
		if channel.IsMember {
			c.sendSlackMessage(text, channel.ID)
		}
	}
}

func (c Connection) sendSlackMessage(text, chanID string) {
	// chanID is either a channel or a group ID
	c.RTM.SendMessage(
		c.RTM.NewOutgoingMessage(text, chanID),
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
	return actions
}

// ServeRTM deals with realtime messages from Slack.
// It is indended to tun in its own goroutine.
func ServeRTM(rtm *slack.RTM) {
	conn := Connection{SlackInfo: nil, RTM: rtm}
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				conn.SlackInfo = ev.Info

			case *slack.MessageEvent:
				stracer.PrettyStruct("Message", ev)
				// from humans (and bots?)
				conn.ParseMessage(ev)

			case *slack.PresenceChangeEvent:
				//stracer.Tracef("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				stracer.Tracef("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				ErrorLogger.Println(ev.Error())

			case *slack.InvalidAuthEvent:
				ErrorLogger.Println("Invalid credentials")
				break

			case *GHPushEvent:
				conn.SendPushMessage(ev)

			case *MessageEvent:
				conn.SendMessage(ev)

			case *NewRelicEvent:
				conn.SendNewRelicMessage(ev)

			default:
				stracer.PrettyStruct("unknown event:", ev)
			}
		}
	}
}
