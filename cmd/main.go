package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		token   string
		chatID  int64
		message string
	)

	flag.StringVar(&token, "token", "", "Telegram Bot Token")
	flag.Int64Var(&chatID, "chat-id", 0, "Telegram Chat ID")
	flag.StringVar(&message, "message", "", "Message to send")
	flag.Parse()

	if token == "" || chatID == 0 || message == "" {
		flag.Usage()
		os.Exit(1)
	}

	/* bot := foxygram.NewBot(token)
	err := bot.SendMessage(chatID, message)
	if err != nil {
		log.Fatal(err)
	} */
	fmt.Println("Message sent successfully")
}
