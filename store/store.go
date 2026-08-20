package store

import (
	"sync"

	"ZaViBiS/arbio/reader"
)

type State struct {
	mu    sync.Mutex
	pairs map[string]reader.OrderBookData
}

func NewState(symbols []string) *State {
	s := &State{
		pairs: make(map[string]reader.OrderBookData),
	}
	for _, symbol := range symbols {
		s.pairs[symbol] = reader.OrderBookData{Symbol: symbol}
	}
	return s
}

func (s *State) Update(symbol string, data reader.OrderBookData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pairs[symbol] = data
}

func (s *State) GetAll() map[string]reader.OrderBookData {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make(map[string]reader.OrderBookData, len(s.pairs))
	for k, v := range s.pairs {
		res[k] = v
	}
	return res
}
