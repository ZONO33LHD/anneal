// Package notify implements the Notifier port (console + Slack).
package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

var badge = map[gateway.NotifyLevel]string{
	gateway.NotifyInfo:     "🔔 INFO",
	gateway.NotifyPriority: "🚨 PRIORITY",
	gateway.NotifyApproval: "🙋 APPROVAL NEEDED",
	gateway.NotifySuccess:  "✅ SUCCESS",
}

// Console prints a Slack-like card to stdout. Default notifier — no secret needed.
type Console struct{}

// NewConsole returns a console notifier.
func NewConsole() gateway.Notifier {
	return Console{}
}

func (Console) Name() string {
	return "console"
}

func (Console) Notify(_ context.Context, msg gateway.NotifyMessage) error {
	var b strings.Builder
	fmt.Fprintf(&b, "\n┌─ %s ─ Anneal\n", badge[msg.Level])
	fmt.Fprintf(&b, "│ %s\n", msg.Title)
	fmt.Fprintf(&b, "│ %s\n", msg.Body)
	if msg.URL != "" {
		fmt.Fprintf(&b, "│ %s\n", msg.URL)
	}
	b.WriteString("└────────────────────────────")
	fmt.Println(b.String())
	return nil
}
