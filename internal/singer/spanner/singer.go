package spanner

import (
	"context"

	"github.com/muzavan/spanner-golang-test/internal/singer"

	googleSpanner "cloud.google.com/go/spanner"
)

// SingerProvider implements singer.Repository
type SingerProvider struct {
	Client *googleSpanner.Client
}

// Create store singer to Spanner
func (repo *SingerProvider) Create(ctx context.Context, payload singer.CreatePayload) error {
	return nil
}

// List fetch singers from Spanner based on filter condition
func (repo *SingerProvider) List(ctx context.Context, filter singer.FilterPayload) ([]singer.Detail, error) {
	return nil, nil
}

// Get fetch singer from Spanner based on its ID
func (repo *SingerProvider) Get(ctx context.Context, singerID int64) (*singer.Detail, error) {
	return nil, nil
}
