package main

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adhocore/gronx"
	"github.com/kyoukaya/endfield-daily/notify"
	"github.com/kyoukaya/endfield-daily/skport"
)

func main() {
	tokens := parseTokens(os.Getenv("ACCOUNT_TOKEN"))
	if len(tokens) == 0 {
		fmt.Fprintln(os.Stderr, "ACCOUNT_TOKEN environment variable is required (one or more tokens separated by newlines)")
		os.Exit(1)
	}

	schedule := os.Getenv("SCHEDULE")

	if schedule == "" {
		if runOnce(tokens) {
			os.Exit(1)
		}
		return
	}

	gron := gronx.New()
	if !gron.IsValid(schedule) {
		fmt.Fprintf(os.Stderr, "Invalid cron expression: %s\n", schedule)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Starting scheduled mode with expression: %s\n", schedule)
	runOnce(tokens)

	for {
		nextTime, err := gronx.NextTick(schedule, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to compute next tick: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Next run at %s\n", nextTime.Format(time.RFC3339))

		delay := time.Until(nextTime)
		timer := time.NewTimer(delay)

		select {
		case <-sig:
			timer.Stop()
			fmt.Println("Received signal, shutting down.")
			return
		case <-timer.C:
			runOnce(tokens)
		}
	}
}

func runOnce(tokens []string) bool {
	discordWebhook := os.Getenv("DISCORD_WEBHOOK")
	discordUser := os.Getenv("DISCORD_USER")

	log := &notify.MessageLog{}

	var notifier notify.Notifier
	if discordWebhook != "" {
		notifier = &notify.Discord{WebhookURL: discordWebhook, UserID: discordUser}
	}

	for i, token := range tokens {
		skport.RunAccount(token, i+1, log)
		if i < len(tokens)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if notifier != nil {
		if err := notifier.Send(log); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send notification: %s\n", err)
		} else {
			fmt.Println("Successfully sent message to Discord webhook!")
		}
	}

	if log.HasError {
		fmt.Fprintln(os.Stderr, "Run completed with errors")
	}
	return log.HasError
}

func parseTokens(raw string) []string {
	var tokens []string
	for _, line := range strings.Split(raw, "\n") {
		decoded, err := url.QueryUnescape(strings.TrimSpace(line))
		if err != nil {
			decoded = strings.TrimSpace(line)
		}
		if decoded != "" {
			tokens = append(tokens, decoded)
		}
	}
	return tokens
}
