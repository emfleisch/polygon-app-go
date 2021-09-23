package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const PRODUCT = "stocks"
const URL = "wss://delayed.polygon.io/" + PRODUCT

const SYMBOL = "AAPL"
const CHANNELS = "T." + SYMBOL

type PolygonTrade struct {
	Ev  string
	Sym string
	I   string
	X   int
	P   float32
	S   int
	C   []int
	T   int64
	Z   int
}

type TradeAggregationRecord struct {
	Sym       string
	Open      float32
	Opentime  int64
	Close     float32
	Closetime int64
	High      float32
	Low       float32
	StartTime int64
	Volume    int
}

func normalizeTime(tradeTime int64) int64 {
	// We want to floor the time window on a 30 second cadence.
	// Ex. 10:01:15 -> 10:01:00, 10:01:42 -> 10:01:30
	t := time.UnixMilli(tradeTime)
	var roundedSecond int = t.Second()
	if roundedSecond < 30 {
		roundedSecond = 0
	} else {
		roundedSecond = 30
	}

	roundedTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), roundedSecond, 0, t.Location())
	roundedUnix := roundedTime.UnixMilli()
	return roundedUnix
}

func processTrades(trade PolygonTrade, tradeData map[int64]TradeAggregationRecord) {
	// Get proper time window for trade, rounding down to nearest 30 second window
	timeWindow := normalizeTime(trade.T)

	// check to see if map is init'd. If not, default it.
	_, ok := tradeData[timeWindow]
	if !ok {
		record := TradeAggregationRecord{StartTime: timeWindow, Sym: SYMBOL, Volume: 0}
		tradeData[timeWindow] = record

	}
	if entry, ok := tradeData[timeWindow]; ok {
		// Check if trade is new high for winow
		if trade.P > entry.High {
			entry.High = trade.P
		}
		// Check if trade is new low for winow
		if trade.P < entry.Low || entry.Low == 0 {
			entry.Low = trade.P
		}
		// Check if trade is opening price for winow
		if trade.T < entry.Opentime || entry.Opentime == 0 {
			entry.Open = trade.P
			entry.Opentime = trade.T
		}
		// Check if trade is closing price for winow
		if trade.T > entry.Closetime {
			entry.Close = trade.P
			entry.Closetime = trade.T

			// Since its at least 30 seconds past a trade, we are out of that trade's window and need to print an update
			currTimeWindow := time.Now()
			endOfCurrTimeWindow := currTimeWindow.Add(time.Second * 30)
			// Also consuming the delayed API, we need to backtrack 15 minutes
			endOfCurrTimeWindow = endOfCurrTimeWindow.Add(-(time.Minute * 15))

			// Only print updates if they are within an hour of the trade time
			endOfUpdateWindow := time.UnixMilli(trade.T).Add(time.Hour + 1).UnixMilli()

			if endOfCurrTimeWindow.UnixMilli() < trade.T && trade.T < endOfUpdateWindow {
				t := time.UnixMilli(timeWindow)
				printData(t, entry)
			}

		}
		// Save for all trades
		entry.Volume += 1
		tradeData[timeWindow] = entry
	}
}

func recieveMessages(wsconn *websocket.Conn, tradeData map[int64]TradeAggregationRecord) {
	for {
		var polygonData []PolygonTrade

		_, msg, err := wsconn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading messages: ")
		}
		// res is byte[], convert to string
		msgStr := string(msg[:])
		if err := json.Unmarshal([]byte(msgStr), &polygonData); err != nil {
			panic(err)
		}
		for _, trade := range polygonData {
			if trade.Ev == "status" {
				fmt.Println("Status message: ", msgStr)
				continue
			}
			processTrades(trade, tradeData)

		}
	}
}

func printData(timestamp time.Time, data TradeAggregationRecord) {
	// Format to look like:
	// 11:36:00 - open: $145.91, close: $145.90, high: $145.91, low: $145.90, volume: 55
	fmt.Printf("%s - open: $%.2f, close: $%.2f, high: $%.2f, low: $%.2f, volume: %d\n", timestamp.Format("15:04:05"), data.Open, data.Close, data.High, data.Low, data.Volume)
}

func printTradeData(tradeData map[int64]TradeAggregationRecord) {

	currtime := time.Now().UTC()
	// Subtracting 15 minutes as we're consuming the delayed API
	lastWindow := currtime.Add(-(time.Second * 30)).Add(-(time.Minute * 15))
	lastWindowUnix := lastWindow.UnixMilli()
	lastWindowUnix = normalizeTime(lastWindowUnix)

	if data, ok := tradeData[lastWindowUnix]; ok {
		printData(time.UnixMilli(lastWindowUnix), data)
	} else {
		printData(time.UnixMilli(lastWindowUnix), TradeAggregationRecord{})
	}

}

func main() {
	// Connect to Websocket
	conn, _, err := websocket.DefaultDialer.Dial(URL, nil)
	if err != nil {
		log.Println("Error connecting to websocket: ", err)
	}

	// Get authtoken from command line argument
	var authtoken string
	flag.StringVar(&authtoken, "a", "", "Enter authtoken")
	flag.Parse()

	// auth the socket
	_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"action\":\"auth\",\"params\":\"%s\"}", authtoken)))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"action\":\"subscribe\",\"params\":\"%s\"}", CHANNELS)))

	// Create trading data storage
	tradeData := make(map[int64]TradeAggregationRecord)

	//read off queue & process messages
	go recieveMessages(conn, tradeData)

	// run forever, print updates every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			printTradeData(tradeData)
		}
	}
}
