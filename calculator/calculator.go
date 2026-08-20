package calculator

import (
	"fmt"
	"strings"
	"time"

	"ZaViBiS/arbio/reader"
	"ZaViBiS/arbio/store"
)

type Action string

const (
	ActionBuy  Action = "BUY"  // Quote -> Base (Amount / Ask)
	ActionSell Action = "SELL" // Base -> Quote (Amount * Bid)
)

type Step struct {
	From   string
	To     string
	Pair   string
	Action Action
}

type Triangle struct {
	Name  string
	Steps [3]Step
}

type StepExecution struct {
	Step      Step
	Price     float64
	AmountIn  float64
	AmountOut float64
	Fee       float64
}

type ArbitrageOpportunity struct {
	Triangle      Triangle
	InitialAmount float64
	FinalAmount   float64
	Profit        float64
	ProfitPercent float64
	FeePercent    float64
	Executions    [3]StepExecution
	Timestamp     time.Time
}

// Common quote currencies on spot markets
var knownQuotes = []string{"USDT", "USDC", "BUSD", "TUSD", "FDUSD", "DAI", "EUR", "BTC", "ETH"}

type PairInfo struct {
	Symbol string
	Base   string
	Quote  string
}

func parsePair(symbol string) (PairInfo, bool) {
	for _, q := range knownQuotes {
		if strings.HasSuffix(symbol, q) && len(symbol) > len(q) {
			return PairInfo{
				Symbol: symbol,
				Base:   symbol[:len(symbol)-len(q)],
				Quote:  q,
			}, true
		}
	}
	return PairInfo{}, false
}

// findTransition checks if a trade from currency `from` to `to` exists
func findTransition(from, to string, pairs map[string]PairInfo) (Step, bool) {
	for _, p := range pairs {
		if p.Quote == from && p.Base == to {
			// Buying Base with Quote: Quote -> Base
			return Step{From: from, To: to, Pair: p.Symbol, Action: ActionBuy}, true
		}
		if p.Base == from && p.Quote == to {
			// Selling Base for Quote: Base -> Quote
			return Step{From: from, To: to, Pair: p.Symbol, Action: ActionSell}, true
		}
	}
	return Step{}, false
}

var currencyPriority = map[string]int{
	"USDT":  100,
	"USDC":  99,
	"FDUSD": 98,
	"DAI":   97,
	"EUR":   96,
	"BTC":   90,
	"ETH":   80,
}

func getPriority(currency string) int {
	if p, ok := currencyPriority[currency]; ok {
		return p
	}
	return 0
}

// BuildTriangles generates all unique 3-step arbitrage cycles without redundant rotations
func BuildTriangles(symbols []string) []Triangle {
	pairs := make(map[string]PairInfo)
	currencySet := make(map[string]struct{})

	for _, s := range symbols {
		if p, ok := parsePair(s); ok {
			pairs[s] = p
			currencySet[p.Base] = struct{}{}
			currencySet[p.Quote] = struct{}{}
		}
	}

	currencies := make([]string, 0, len(currencySet))
	for c := range currencySet {
		currencies = append(currencies, c)
	}

	var triangles []Triangle
	n := len(currencies)

	// Iterate over unique triplets of currencies (i < j < k)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			for k := j + 1; k < n; k++ {
				triplet := []string{currencies[i], currencies[j], currencies[k]}

				// Select canonical starting currency (highest priority, or alphabetical)
				root := triplet[0]
				for _, c := range triplet[1:] {
					if getPriority(c) > getPriority(root) || (getPriority(c) == getPriority(root) && c < root) {
						root = c
					}
				}

				// The other two currencies
				var others []string
				for _, c := range triplet {
					if c != root {
						others = append(others, c)
					}
				}

				a, b := others[0], others[1]

				// Direction 1: root -> a -> b -> root
				if s1, ok1 := findTransition(root, a, pairs); ok1 {
					if s2, ok2 := findTransition(a, b, pairs); ok2 {
						if s3, ok3 := findTransition(b, root, pairs); ok3 {
							triangles = append(triangles, Triangle{
								Name:  fmt.Sprintf("%s -> %s -> %s -> %s", root, a, b, root),
								Steps: [3]Step{s1, s2, s3},
							})
						}
					}
				}

				// Direction 2: root -> b -> a -> root
				if s1, ok1 := findTransition(root, b, pairs); ok1 {
					if s2, ok2 := findTransition(b, a, pairs); ok2 {
						if s3, ok3 := findTransition(a, root, pairs); ok3 {
							triangles = append(triangles, Triangle{
								Name:  fmt.Sprintf("%s -> %s -> %s -> %s", root, b, a, root),
								Steps: [3]Step{s1, s2, s3},
							})
						}
					}
				}
			}
		}
	}

	return triangles
}

// EvaluateTriangle calculates the outcome of executing a triangular trade taking fees into account.
func EvaluateTriangle(tri Triangle, orderBooks map[string]reader.OrderBookData, initialAmount float64, feePercent float64) (*ArbitrageOpportunity, bool) {
	currentAmount := initialAmount
	var executions [3]StepExecution

	feeRate := feePercent / 100.0

	for i, step := range tri.Steps {
		ob, ok := orderBooks[step.Pair]
		if !ok || len(ob.Bids) == 0 || len(ob.Asks) == 0 {
			return nil, false
		}

		bestBid := ob.Bids[0].Price
		bestAsk := ob.Asks[0].Price

		var price float64
		var rawAmountOut float64

		if step.Action == ActionBuy {
			if bestAsk <= 0 {
				return nil, false
			}
			price = bestAsk
			rawAmountOut = currentAmount / price
		} else { // ActionSell
			if bestBid <= 0 {
				return nil, false
			}
			price = bestBid
			rawAmountOut = currentAmount * price
		}

		fee := rawAmountOut * feeRate
		amountOut := rawAmountOut - fee

		executions[i] = StepExecution{
			Step:      step,
			Price:     price,
			AmountIn:  currentAmount,
			AmountOut: amountOut,
			Fee:       fee,
		}
		currentAmount = amountOut
	}

	profit := currentAmount - initialAmount
	profitPercent := (profit / initialAmount) * 100.0

	return &ArbitrageOpportunity{
		Triangle:      tri,
		InitialAmount: initialAmount,
		FinalAmount:   currentAmount,
		Profit:        profit,
		ProfitPercent: profitPercent,
		FeePercent:    feePercent,
		Executions:    executions,
		Timestamp:     time.Now(),
	}, true
}

func Start(state *store.State, notify <-chan struct{}) {
	fmt.Println("[CALCULATOR] Запуск калькулятора трикутного арбітражу...")

	initialData := state.GetAll()
	var symbols []string
	for s := range initialData {
		symbols = append(symbols, s)
	}

	triangles := BuildTriangles(symbols)
	fmt.Printf("[CALCULATOR] Знайдено %d арбітражних трикутників для моніторингу:\n", len(triangles))
	for _, tri := range triangles {
		fmt.Printf("  • %s\n", tri.Name)
	}
	fmt.Println("[CALCULATOR] Очікування оновлень склянки...")

	const initialCapital = 100.0 // Умовний стартовий баланс для тесту
	const feePercent = 0.01      // Комісія 0.1% на кожну угоду

	for range notify {
		data := state.GetAll()

		for _, tri := range triangles {
			opp, ok := EvaluateTriangle(tri, data, initialCapital, feePercent)
			if !ok {
				continue
			}

			// Виводимо тільки якщо є чистий прибуток після комісій
			if opp.Profit > 0 {
				printOpportunity(opp)
			}
		}
	}
}

func printOpportunity(opp *ArbitrageOpportunity) {
	fmt.Printf("\n======================================================================\n")
	fmt.Printf("💰 [ПРИБУТОК ЗНАЙДЕНО] %s\n", opp.Timestamp.Format("15:04:05.000"))
	fmt.Printf("Трикутник:         %s\n", opp.Triangle.Name)
	fmt.Printf("Початковий баланс: %.4f %s\n", opp.InitialAmount, opp.Triangle.Steps[0].From)
	fmt.Printf("Кінцевий баланс:   %.4f %s\n", opp.FinalAmount, opp.Triangle.Steps[0].From)
	fmt.Printf("Прибуток (чистий): +%.4f %s (+%.4f%%)\n", opp.Profit, opp.Triangle.Steps[0].From, opp.ProfitPercent)
	fmt.Printf("Комісія:           %.2f%% на кожну угоду\n", opp.FeePercent)
	fmt.Println("Кроки угоди:")
	for i, ex := range opp.Executions {
		var actionStr string
		if ex.Step.Action == ActionBuy {
			actionStr = fmt.Sprintf("BUY  %s за %s (Ask: %.8f)", ex.Step.To, ex.Step.From, ex.Price)
		} else {
			actionStr = fmt.Sprintf("SELL %s на %s (Bid: %.8f)", ex.Step.From, ex.Step.To, ex.Price)
		}
		fmt.Printf("  %d. %s [%s]\n     Вхід: %.6f %s => Вихід: %.6f %s (Комісія: %.6f %s)\n",
			i+1, actionStr, ex.Step.Pair, ex.AmountIn, ex.Step.From, ex.AmountOut, ex.Step.To, ex.Fee, ex.Step.To)
	}
	fmt.Printf("======================================================================\n\n")
}
