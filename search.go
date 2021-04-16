package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/antoan-angelov/go-fuzzy"
	"github.com/bwmarrin/discordgo"
	"robpike.io/filter"
)

var (
	topFuzz        *fuzzy.Fuzzy
	libTopFuzz     *fuzzy.Fuzzy
	lopPropFuzz    *fuzzy.Fuzzy
	libTopPropFuzz *fuzzy.Fuzzy

	questionMatch   *regexp.Regexp
	prefixMatch     *regexp.Regexp
	topMatch        *regexp.Regexp
	libTopMatch     *regexp.Regexp
	topPropMatch    *regexp.Regexp
	libTopPropMatch *regexp.Regexp
)

func init() {
	questionMatch = regexp.MustCompile(`([?!&])\[(.*?)\]`)
	prefixMatch = regexp.MustCompile(`^([A-Za-z$_][A-Za-z0-9$_:.]*?\.)([A-Za-z$_A-Za-z0-9$_]*?)\.([A-Za-z$_<+|[>\/^~&*%=\-][A-Za-z0-9$_\]=\/<>\-]*?)$`)
	topMatch = regexp.MustCompile(`^([A-Za-z$_A-Za-z0-9$_]*?)$`)
	libTopMatch = regexp.MustCompile(`^([A-Za-z$_][A-Za-z0-9$_:.]*?)\/([A-Za-z$_A-Za-z0-9$_]*?)$`)
	topPropMatch = regexp.MustCompile(`^([A-Za-z$_A-Za-z0-9$_]*?)\.([A-Za-z$_<+|[>\/^~&*%=\-][A-Za-z0-9$_\]=\/<>\-]*?)$`)
	libTopPropMatch = regexp.MustCompile(`^([A-Za-z$_][A-Za-z0-9$_:.]*?)\/([A-Za-z$_A-Za-z0-9$_]*?)\.([A-Za-z$_<+|[>\/^~&*%=\-][A-Za-z0-9$_\]=\/<>\-]*?)$`)
}

// Search for an element in the documentation or on pub
func Search(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.Bot {
		return
	}

	if len(message.Content) < 4 {
		return
	}

	query := message.Content

	matches := questionMatch.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		switch match[1] {
		case `!`:
			fallthrough
		case `?`:
			out := []interface{}{}
			var err error
			if topMatch.FindString(match[2]) != "" {
				out, err = topFuzz.Search(match[2])
			} else if topPropMatch.FindString(match[2]) != "" {
				out, err = lopPropFuzz.Search(match[2])
			} else if libTopMatch.FindString(match[2]) != "" {
				out, err = libTopFuzz.Search(strings.Replace(match[2], "/", ".", 1))
			} else if libTopPropMatch.FindString(match[2]) != "" {
				out, err = libTopPropFuzz.Search(strings.Replace(match[2], "/", ".", 1))
			}

			if err != nil || len(out) == 0 {
				notFound(session, message.ChannelID, match[2])
				return
			}

			if match[1] == "!" {
				session.ChannelMessageSend(
					message.ChannelID,
					"https://api.flutter.dev/flutter/"+out[0].(SearchStructElement).Href,
				)
			} else {
				fields := []*discordgo.MessageEmbedField{}

				for _, result := range out[0:min(10, len(out))] {
					if result.(SearchStructElement).EnclosedBy != nil {
						fields = append(fields, &discordgo.MessageEmbedField{
							Name:  result.(SearchStructElement).Type + " " + result.(SearchStructElement).Name + " - " + result.(SearchStructElement).EnclosedBy.Name,
							Value: "https://api.flutter.dev/flutter/" + result.(SearchStructElement).Href,
						})
					} else {
						fields = append(fields, &discordgo.MessageEmbedField{
							Name:  result.(SearchStructElement).Type + " " + result.(SearchStructElement).Name,
							Value: "https://api.flutter.dev/flutter/" + result.(SearchStructElement).Href,
						})
					}

				}

				session.ChannelMessageSendEmbed(
					message.ChannelID,
					&discordgo.MessageEmbed{
						Title:  "Pub Search Results - " + match[2],
						Fields: fields,
					},
				)
			}
		case `&`:
			r, err := http.Get("https://pub.dev/api/search?q=" + match[2])
			if err != nil {
				return
			}
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return
			}
			s, err := unmarshalPubSearch(b)
			if err != nil {
				return
			}
			if len(s.Packages) == 0 {
				notFound(session, message.ChannelID, match[2])
				return
			}
			fields := []*discordgo.MessageEmbedField{}

			for _, result := range s.Packages[0:min(10, len(s.Packages))] {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  result.Package,
					Value: "https://pub.dev/packages/" + result.Package,
				})
			}

			session.ChannelMessageSendEmbed(
				message.ChannelID,
				&discordgo.MessageEmbed{
					Title:  "Pub Search Results - " + match[2],
					Fields: fields,
				},
			)
		}
	}
}

func notFound(s *discordgo.Session, channel string, message string) {
	s.ChannelMessageSendEmbed(
		channel,
		&discordgo.MessageEmbed{
			Title:       "Not Found",
			Color:       0xDD2222,
			Description: message,
		},
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func updateCache() {
	r, err := http.Get("https://api.flutter.dev/flutter/index.json")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	s, err := unmarshalSearchStruct(b)
	if err != nil {
		panic(err)
	}

	top := filter.Choose(s, func(value interface{}) bool {
		return (value.(SearchStructElement).EnclosedBy != nil && value.(SearchStructElement).EnclosedBy.Type == "library")
	}).([]interface{})

	libTopProp := filter.Choose(s, func(value interface{}) bool {
		return (value.(SearchStructElement).EnclosedBy != nil && value.(SearchStructElement).EnclosedBy.Type != "library")
	}).([]interface{})

	topProp := filter.Apply(libTopProp, func(value interface{}) interface{} {
		test := prefixMatch.FindStringSubmatch(value.(SearchStructElement).QualifiedName)
		if len(test) == 0 {
			fmt.Printf("Did not find this %v\n", value)
			panic("a")
		}
		return SearchStructElement{
			EnclosedBy:      value.(SearchStructElement).EnclosedBy,
			QualifiedName:   strings.TrimPrefix(value.(SearchStructElement).QualifiedName, test[1]),
			Href:            value.(SearchStructElement).Href,
			Name:            value.(SearchStructElement).Name,
			OverriddenDepth: value.(SearchStructElement).OverriddenDepth,
			Type:            value.(SearchStructElement).Type,
		}
	}).([]interface{})

	topFuzz = toFuzz(&top, "Name")
	libTopFuzz = toFuzz(&top, "QualifiedName")
	lopPropFuzz = toFuzz(&topProp, "QualifiedName")
	libTopPropFuzz = toFuzz(&libTopProp, "QualifiedName")
}

func toFuzz(elements *[]interface{}, key string) (f *fuzzy.Fuzzy) {
	f = fuzzy.NewFuzzy()
	f.Set(elements)
	f.Options.SetThreshold(5)
	f.SetKeys([]string{key})
	return
}
