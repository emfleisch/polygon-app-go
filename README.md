## Polygon Ticker Trading Aggregation App

This project consumes the Polygon.io delayed stocks API and aggregates prices for AAPL stock based on 30 second intervals. 
Note that because the delayed API is being used, all trades are 15 minutes old. This will be reflected in the time window that is printed. 

The output will be the last 30 second window's information including:
- Window Start Time
- Opening Price
- Closing Price
- High Price for window
- Low Price for window
- Volume of trades in that window

With the output format being:
> 11:36:00 - open: $145.91, close: $145.90, high: $145.91, low: $145.90, volume: 55


### To Start The Project
- Have go version 1.17 installed ()
-- For assitance, visit https://golang.org/doc/install
- Run "go get" to install deps from go.mod file
- Get a auth token for the Polygon.io Delayed Stocks Websocket API
- Execute the app with the follow line:
```
go run main.go -a <Auth Token Here>
```
