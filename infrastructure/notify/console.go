// Package notify は Notifier のポートを実装する（console + Slack）。
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

// Console は Slack 風のカードを stdout へ出力する。デフォルトの Notifier で、シークレットは不要。
type Console struct{}

// NewConsole はコンソールの Notifier を返す。
func NewConsole() gateway.Notifier {
	return Console{}
}

func (Console) ProviderName() string {
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
