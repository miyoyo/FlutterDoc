package main

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

var messageCache = map[string][]*discordgo.Message{}
var metaChannel = "421444762956988418"
var rulesChannel = "1137753387715264613"
var emojiRegex = regexp.MustCompile(`<(:.*?:)\d{19}>`)

// DeDupe messages sent on the server by caching them into a map and comparing them as they come in
func DeDupe(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.GuildID == "" {
		// DMs do not count
		return
	}

	if m.Author.Bot {
		return
	}

	if len(messageCache) == 0 {
		// On start, drill all channels once
		// Cache 5 messages deep
		for _, v := range s.State.Guilds[0].Channels {
			fmt.Println(v.Name)
			messageCache[v.ID], _ = s.ChannelMessages(v.ID, 5, "", "", "")
		}
		return
	}

	filtered := emojiRegex.ReplaceAllString(m.ContentWithMentionsReplaced(), "$1")

	if len(filtered) >= 20 {
		sent := false
		count := 0
	channelLoop:
		for k, c := range messageCache {
			if k == m.ChannelID {
				continue
			}
			for _, v := range c {
				if m.Content == v.Content && m.Author.ID == v.Author.ID {
					if !sent {
						s.ChannelMessageSend(metaChannel, "Hey, "+m.Author.Mention()+", please take a second to read the "+fmt.Sprintf("<#%s>", rulesChannel)+",\nspecifically, the section about not duplicating your messages across channels.\nIf you want to move a message, copy it, delete it, **then** paste it in another channel.\n\nThanks!")
						sent = true
					}
					count++
				}

				// Ban the user after 6 messages: Origin + 5 additional messages
				if count >= 5 {
					s.GuildBanCreateWithReason(s.State.Guilds[0].ID, m.Author.ID, "Excessive spam", 7)
					break channelLoop
				}
			}
		}
	}

	if len(messageCache[m.ChannelID]) == 5 {
		messageCache[m.ChannelID] = messageCache[m.ChannelID][:4]
	}

	messageCache[m.ChannelID] = append([]*discordgo.Message{m.Message}, messageCache[m.ChannelID]...)

}

// DeleteDeDupe or rather mask deleted messages
func DeleteDeDupe(s *discordgo.Session, m *discordgo.MessageDelete) {
	for _, v := range messageCache[m.ChannelID] {
		v.Author = s.State.User
		v.Content = ""
	}
}
