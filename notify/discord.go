package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Discord sends notifications via a Discord webhook.
type Discord struct {
	WebhookURL string
	UserID     string
}

// Send posts the message log to Discord.
func (d *Discord) Send(log *MessageLog) error {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(d.WebhookURL)), "https://discord.com/api/webhooks/") {
		return fmt.Errorf("invalid Discord webhook URL")
	}

	var b strings.Builder
	if d.UserID != "" {
		fmt.Fprintf(&b, "<@%s>\n", d.UserID)
	}
	b.WriteString("**Endfield Daily Check-in**\n")
	for _, msg := range log.Messages {
		fmt.Fprintf(&b, "(%s) %s\n", strings.ToUpper(msg.Level), msg.Text)
	}

	payload, _ := json.Marshal(map[string]string{"content": b.String()})
	resp, err := http.Post(d.WebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("discord webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return nil
}
