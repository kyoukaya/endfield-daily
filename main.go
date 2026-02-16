package main

import (
	"fmt"
	"net/http"
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

	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go func() {
		fmt.Printf("Health check listening on :%s\n", healthPort)
		if err := http.ListenAndServe(":"+healthPort, mux); err != nil {
			fmt.Fprintf(os.Stderr, "Health server error: %s\n", err)
		}
	}()

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
	notifyNoOps := os.Getenv("NOTIFY_NO_OPS") != ""

	var notifier notify.Notifier
	if discordWebhook != "" {
		notifier = &notify.Discord{WebhookURL: discordWebhook, UserID: discordUser}
	}

	hasError := false
	for i, token := range tokens {
		if err := skport.RunAccount(token, i+1, notifier, notifyNoOps); err != nil {
			hasError = true
		}
		if i < len(tokens)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if hasError {
		fmt.Fprintln(os.Stderr, "Run completed with errors")
	}
	return hasError
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
