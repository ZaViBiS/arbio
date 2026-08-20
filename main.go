package main

import (
	"ZaViBiS/arbio/calculator"
	"ZaViBiS/arbio/reader"
	"ZaViBiS/arbio/store"
)

func main() {
	symbols := []string{"BTCUSDT", "ETHUSDT", "ETHBTC", "SOLUSDT", "SOLBTC"}

	notify := make(chan struct{}, 1)

	state := store.NewState(symbols)

	go reader.BybitWorker(symbols, state, notify)

	calculator.Start(state, notify)
}
