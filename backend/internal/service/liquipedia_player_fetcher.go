package service

import (
	"context"
	"fmt"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// LiquipediaPlayerFetcher loads player pages via Liquipedia HTML parse.
type LiquipediaPlayerFetcher struct {
	Client *liquipedia.Client
}

func (f LiquipediaPlayerFetcher) FetchPlayerPage(ctx context.Context, link string) (model.PlayerPage, error) {
	if f.Client == nil {
		return model.PlayerPage{}, fmt.Errorf("liquipedia client is nil")
	}
	canonical, err := liquipedia.ValidateURL(link)
	if err != nil {
		return model.PlayerPage{}, err
	}
	fetched, err := f.Client.FetchPage(ctx, canonical)
	if err != nil {
		return model.PlayerPage{}, err
	}
	return parse.Player(canonical, fetched.HTML)
}
