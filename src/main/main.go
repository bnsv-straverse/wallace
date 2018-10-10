package main

import (
	"fmt"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	fmt.Println("Running wallace v0.0.2")
	api := slack.New(os.Getenv("API_KEY"))

	cmdManager := CommandManager{}
	registerCommands(&cmdManager)

	fmt.Printf("Registered %d commands\n", cmdManager.getCommandCount())

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			cmdManager.execute(rtm, ev)
		}
	}
}
