package mexc

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const url = "wss://wbs.mexc.com/ws"

// Структура для відправки запиту на підписку (формат MEXC)
type SubscribeMsg struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func Test() {
	log.Printf("Підключення до %s...", url)

	headers := http.Header{}
	headers.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0")

	// 2. Встановлюємо з'єднання
	// DefaultDialer підходить для більшості випадків
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		log.Fatalf("Помилка підключення: %v", err)
	}
	// Важливо: закриваємо з'єднання при виході з функції
	defer conn.Close()

	log.Println("Успішно підключено!")

	// 3. Формуємо повідомлення для підписки
	// Формат згідно з документацією MEXC для публічного каналу
	subMsg := SubscribeMsg{
		Method: "SUBSCRIPTION",
		Params: []string{"spot@public.bookTicker.v3.api@BTCUSDT"}, // Канал найкращої ціни Bid/Ask для BTC
	}

	// 4. Відправляємо повідомлення (серіалізуємо структуру в JSON)
	err = conn.WriteJSON(subMsg)
	if err != nil {
		log.Fatalf("Помилка відправки підписки: %v", err)
	}
	log.Println("Запит на підписку відправлено")

	// 5. Запускаємо безкінечний цикл читання вхідних повідомлень
	// (у реальному проекті це краще робити в окремій горутині)
	for {
		// ReadMessage повертає тип повідомлення (текст/бінарне) і самі байти
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Помилка читання (можливо, розрив з'єднання):", err)
			return // Виходимо з циклу при розриві
		}

		// Перевіряємо, чи це текстове повідомлення
		if messageType == websocket.TextMessage {
			// Для простоти виводимо сирий JSON як рядок
			// У реальному коді тут буде json.Unmarshal у структуру
			fmt.Printf("Отримано дані: %s\n", string(message))
		}

		// Робимо невелику паузу, щоб не спамити консоль, хоча це не обов'язково
		time.Sleep(100 * time.Millisecond)
	}
}
