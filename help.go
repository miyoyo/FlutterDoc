package main

import "github.com/bwmarrin/discordgo"

// Help users when they mention the bot
func Help(s *discordgo.Session, h *discordgo.MessageCreate) {
	for _, user := range h.Mentions {
		if user.ID == s.State.User.ID {
			s.ChannelMessageSendEmbed(h.ChannelID, &discordgo.MessageEmbed{
				Title:       "ℹ️ Help",
				Description: "⚠️ These commands can be within a message, and there can be multiple per messages",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "![Object] or ![Object.property] or ![package/Object] or ![package/Object.property]",
						Value: "Gives a direct link to the closest match from the flutter documentation",
					},
					{
						Name:  "?[Object] or ?[Object.property]  or ?[package/Object] or ?[package/Object.property]",
						Value: "Shows the 10 first search results from the flutter documentation",
					},
					{
						Name:  "&[package]",
						Value: "Shows up to 10 search results about 'package' on Pub",
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Source: https://gist.github.com/miyoyo/8641057636892863791ca7c41a1fab97",
				},
			})
		}
	}
}
