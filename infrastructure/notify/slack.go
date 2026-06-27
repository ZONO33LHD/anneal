package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

var emoji = map[gateway.NotifyLevel]string{
	gateway.NotifyInfo:     ":bell:",
	gateway.NotifyPriority: ":rotating_light:",
	gateway.NotifyApproval: ":raising_hand:",
	gateway.NotifySuccess:  ":white_check_mark:",
}

// Slack posts to an Incoming Webhook using the standard HTTP client.
type Slack struct {
	webhookURL string
	http       *http.Client
}

// NewSlack builds a Slack notifier.
func NewSlack(webhookURL string) gateway.Notifier {
	return &Slack{webhookURL: webhookURL, http: &http.Client{Timeout: 10 * time.Second}}
}

func (Slack) Name() string { return "slack" }

func (s *Slack) Notify(ctx context.Context, msg gateway.NotifyMessage) error {
	lines := []string{fmt.Sprintf("%s *%s*", emoji[msg.Level], msg.Title), msg.Body}
	if msg.URL != "" {
		lines = append(lines, fmt.Sprintf("<%s|Open>", msg.URL))
	}
	payload, _ := json.Marshal(map[string]string{"text": strings.Join(lines, "\n")})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
