package hyperliquid_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/koative/hyperliquid-go"
)

func ExampleInfoClient_L2Book() {
	info := hyperliquid.NewInfoClient(hyperliquid.Mainnet)
	book, err := info.L2Book(context.Background(), hyperliquid.L2BookRequest{Coin: "BTC"})
	if err != nil {
		log.Fatal(err)
	}
	bids, asks := book.Levels[0], book.Levels[1]
	fmt.Printf("BTC %s / %s\n", bids[0].Px, asks[0].Px)
}

func ExampleExchangeClient_Order() {
	// Trade with an API (agent) wallet approved for the account.
	signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, signer)

	statuses, err := ex.Order(context.Background(), hyperliquid.OrderAction{
		Orders: []hyperliquid.Order{{
			Asset: 0, // BTC
			IsBuy: true,
			Price: "50000",
			Size:  "0.001",
			Type:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifGtc}},
		}},
	})
	var rejected *hyperliquid.StatusError
	switch {
	case errors.As(err, &rejected):
		log.Printf("order %d rejected: %s", rejected.Index, rejected.Message)
	case err != nil:
		log.Fatal(err)
	case statuses[0].Resting != nil:
		fmt.Println("resting oid:", statuses[0].Resting.Oid)
	case statuses[0].Filled != nil:
		fmt.Println("filled at", statuses[0].Filled.AvgPx)
	}
}

func ExampleFormatPrice() {
	// ETH perp has szDecimals 4: at most 5 significant figures and 2 decimals.
	px, _ := hyperliquid.FormatPrice("2712.3456", 4, false)
	sz, _ := hyperliquid.FormatSize("0.123456", 4)
	fmt.Println(px, sz)
	// Output: 2712.3 0.1234
}

func ExampleNewPrivateKeySigner() {
	signer, err := hyperliquid.NewPrivateKeySigner("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(signer.Address())
	// Output: 0x14791697260e4c9a71f18484c9f997b308e59325
}
