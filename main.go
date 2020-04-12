package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonlvhit/gocron"
)

func main() {
	fmt.Print("Loading search structure...")
	updateCache()
	fmt.Println(" Done")

	bot, err := discordgo.New("Bot TOKEN")
	if err != nil {
		panic("Could not create bot: " + err.Error())
	}

	bot.AddHandler(Search)

	if bot.Open() != nil {
		panic("Could not open bot: " + err.Error())
	}

	go func() {
		gocron.Every(1).Day().Do(func() {
			bot.UpdateStatusComplex(discordgo.UpdateStatusData{
				Status: "idle",
				AFK:    true,
				Game: &discordgo.Game{
					Name: "updating search...",
				},
			})
			updateCache()
			bot.UpdateStatus(0, "mention me for commands.")
		})
		gocron.Every(30).Minutes().Do(func() { bot.UpdateStatus(0, "mention me for commands.") })
	}()

	fmt.Println("FlutterDoc running. CTRL+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	bot.Close()
}
