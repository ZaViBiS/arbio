package calculator

import (
	"testing"

	"ZaViBiS/arbio/reader"
)

func TestBuildTriangles(t *testing.T) {
	symbols := []string{"BTCUSDT", "ETHUSDT", "ETHBTC"}
	triangles := BuildTriangles(symbols)

	if len(triangles) != 2 {
		t.Fatalf("expected 2 triangles, got %d", len(triangles))
	}
}

func TestEvaluateTriangle_Profit(t *testing.T) {
	// 1 USDT -> BTC -> ETH -> USDT
	// Step 1: Buy BTC with USDT @ 60,000 => 100 / 60000 = 0.001666666 BTC
	// Step 2: Buy ETH with BTC @ 0.05 => 0.001666666 / 0.05 = 0.033333333 ETH
	// Step 3: Sell ETH for USDT @ 3100 => 0.033333333 * 3100 = 103.33333 USDT (+3.33%)

	tri := Triangle{
		Name: "USDT -> BTC -> ETH -> USDT",
		Steps: [3]Step{
			{From: "USDT", To: "BTC", Pair: "BTCUSDT", Action: ActionBuy},
			{From: "BTC", To: "ETH", Pair: "ETHBTC", Action: ActionBuy},
			{From: "ETH", To: "USDT", Pair: "ETHUSDT", Action: ActionSell},
		},
	}

	orderBooks := map[string]reader.OrderBookData{
		"BTCUSDT": {
			Symbol: "BTCUSDT",
			Bids:   []reader.Level{{Price: 59990, Qty: 1}},
			Asks:   []reader.Level{{Price: 60000, Qty: 1}},
		},
		"ETHBTC": {
			Symbol: "ETHBTC",
			Bids:   []reader.Level{{Price: 0.049, Qty: 10}},
			Asks:   []reader.Level{{Price: 0.050, Qty: 10}},
		},
		"ETHUSDT": {
			Symbol: "ETHUSDT",
			Bids:   []reader.Level{{Price: 3100, Qty: 10}},
			Asks:   []reader.Level{{Price: 3110, Qty: 10}},
		},
	}

	opp, ok := EvaluateTriangle(tri, orderBooks, 100.0)
	if !ok {
		t.Fatalf("expected opportunity calculation to succeed")
	}

	if opp.Profit <= 0 {
		t.Fatalf("expected profit > 0, got %f", opp.Profit)
	}

	expectedFinal := 100.0 / 60000.0 / 0.050 * 3100.0
	if opp.FinalAmount != expectedFinal {
		t.Fatalf("expected final amount %f, got %f", expectedFinal, opp.FinalAmount)
	}
}

func TestEvaluateTriangle_Loss(t *testing.T) {
	tri := Triangle{
		Name: "USDT -> BTC -> ETH -> USDT",
		Steps: [3]Step{
			{From: "USDT", To: "BTC", Pair: "BTCUSDT", Action: ActionBuy},
			{From: "BTC", To: "ETH", Pair: "ETHBTC", Action: ActionBuy},
			{From: "ETH", To: "USDT", Pair: "ETHUSDT", Action: ActionSell},
		},
	}

	orderBooks := map[string]reader.OrderBookData{
		"BTCUSDT": {
			Symbol: "BTCUSDT",
			Bids:   []reader.Level{{Price: 59990, Qty: 1}},
			Asks:   []reader.Level{{Price: 60000, Qty: 1}},
		},
		"ETHBTC": {
			Symbol: "ETHBTC",
			Bids:   []reader.Level{{Price: 0.049, Qty: 10}},
			Asks:   []reader.Level{{Price: 0.050, Qty: 10}},
		},
		"ETHUSDT": {
			Symbol: "ETHUSDT",
			Bids:   []reader.Level{{Price: 2900, Qty: 10}}, // Low bid -> loss
			Asks:   []reader.Level{{Price: 2910, Qty: 10}},
		},
	}

	opp, ok := EvaluateTriangle(tri, orderBooks, 100.0)
	if !ok {
		t.Fatalf("expected evaluation to succeed")
	}

	if opp.Profit >= 0 {
		t.Fatalf("expected loss, got profit: %f", opp.Profit)
	}
}
