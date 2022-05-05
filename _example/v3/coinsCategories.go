package main

import (
	"context"
	"log"
	"time"

	gecko "github.com/Grishameister/go-gecko/v3"
	geckoTypes "github.com/Grishameister/go-gecko/v3/types"
)

func main() {
	t := time.Time{}
	startHook := func() {
		t = time.Now()
	}

	endHook := func() {
		log.Println(time.Since(t))
	}

	cg := gecko.NewClient(gecko.WithStartHook(startHook), gecko.WithEndHook(endHook))

	ctx := context.Background()
	// find specific coins

	res, err := cg.CoinsCategories(ctx, geckoTypes.OrderTypeObject.MarketCapDesc)
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range res {
		if c != nil {
			log.Println(*c)
		}
	}
}
