// Package firestore は Firestore を使って repository ポートを実装する。
package firestore

import (
	"context"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// NewClient は Firestore クライアントを生成する。
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*cloudfirestore.Client, error) {
	return cloudfirestore.NewClient(ctx, projectID, opts...)
}
