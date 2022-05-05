package main

import (
	"context"
	"log"
	"sync"
	"time"

	gecko "github.com/Grishameister/go-gecko/v3"
	geckoTypes "github.com/Grishameister/go-gecko/v3/types"
)

func main() {
	t := time.Time{}
	allTiming := time.Now()
	startHook := func() {
		t = time.Now()
	}

	endHook := func() {
		log.Println(time.Since(t))
	}

	cg := gecko.NewClient(gecko.WithStartHook(startHook), gecko.WithEndHook(endHook))

	ctx := context.Background()
	// find specific coins
	vsCurrency := "usd"
	perPage := 250
	// page := 3
	sparkline := true
	pcp := geckoTypes.PriceChangePercentageObject
	priceChangePercentage := []string{pcp.PCP1h, pcp.PCP24h, pcp.PCP7d, pcp.PCP14d, pcp.PCP30d, pcp.PCP200d, pcp.PCP1y}
	order := geckoTypes.OrderTypeObject.MarketCapDesc

	marketChan := make(chan *geckoTypes.CoinsMarketItem)
	wg := sync.WaitGroup{}

	for i := 1; i < 9; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			market, err := cg.CoinsMarket(ctx, vsCurrency, nil, order, perPage, i, sparkline, priceChangePercentage)
			if err != nil {
				log.Fatal(err)
			}

			for _, m := range market {
				marketChan <- m
			}
		}()
	}

	go func() {
		defer close(marketChan)
		wg.Wait()
	}()

	counter := 0
	for it := range marketChan {
		log.Println(it)
		counter++
	}

	log.Println(time.Since(allTiming), counter)
}
