package reader

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const url = "wss://stream.bybit.com/v5/public/spot"

type SubMessage struct {
	OP   string   `json:"op"`
	Args []string `json:"args"`
}

type OrderBookMessage struct {
	Topic string        `json:"topic"`
	TS    int64         `json:"ts"`
	Type  string        `json:"type"`
	Data  OrderBookData `json:"data"`
	CTS   int64         `json:"cts"`
}

type OrderBookData struct {
	Symbol   string  `json:"s"`
	Bids     []Level `json:"b"`
	Asks     []Level `json:"a"`
	UpdateID int64   `json:"u"`
	Seq      int64   `json:"seq"`
}

type StateUpdate interface {
	Update(symbol string, data OrderBookData)
}

// UnmarshalJSON converts a Bybit level such as ["0.07736", "97"]
// into a typed value.
func (l *Level) UnmarshalJSON(data []byte) error {
	var raw [2]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	price, err := strconv.ParseFloat(raw[0], 64)
	if err != nil {
		return fmt.Errorf("invalid price %q: %w", raw[0], err)
	}
	qty, err := strconv.ParseFloat(raw[1], 64)
	if err != nil {
		return fmt.Errorf("invalid quantity %q: %w", raw[1], err)
	}

	l.Price = price
	l.Qty = qty
	return nil
}

func BybitWorker(symbols []string, state StateUpdate, notify chan<- struct{}) error {
	log.Printf("Підключення до %s...", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("помилка підключення: %w", err)
	}
	defer conn.Close()

	log.Println("Успішно підключено!")

	var args []string
	for _, s := range symbols {
		args = append(args, "orderbook.1."+s)
	}

	subMsg := SubMessage{
		OP:   "subscribe",
		Args: args,
	}

	err = conn.WriteJSON(subMsg)
	if err != nil {
		return fmt.Errorf("помилка відправки підписки: %w", err)
	}
	log.Println("Запит на підписку відправлено")

	// Bybit closes idle public WebSocket connections without periodic pings.
	stopPing := make(chan struct{})
	defer close(stopPing)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteJSON(map[string]string{"op": "ping"}); err != nil {
					return
				}
			case <-stopPing:
				return
			}
		}
	}()

	for {
		// ReadMessage повертає тип повідомлення (текст/бінарне) і самі байти
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Помилка читання (можливо, розрив з'єднання):", err)
			return err
		}

		// Перевіряємо, чи це текстове повідомлення
		if messageType == websocket.TextMessage {
			var orderBook OrderBookMessage
			if err := json.Unmarshal(message, &orderBook); err != nil {
				log.Printf("Помилка розбору order book: %v", err)
				continue
			}
			if orderBook.Topic == "" {
				log.Printf("Bybit service message: %s", string(message))
				continue
			}
			if orderBook.Topic != "" {
				//fmt.Println(orderBook)
				state.Update(orderBook.Data.Symbol, orderBook.Data)
				notify <- struct{}{}
			}
		}
	}
}
