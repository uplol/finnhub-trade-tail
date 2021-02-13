package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/urfave/cli"
)

const (
	FINNHUB_STREAM_URL = "wss://ws.finnhub.io?token=%s"
)

func run(ctx *cli.Context) error {
	token := ctx.String("token")
	if token == "" {
		return errors.New("must provide a token")
	}

	c, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf(FINNHUB_STREAM_URL, token), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				return
			}

			if messageType != websocket.TextMessage {
				continue
			}

			var data map[string]interface{}
			err = json.Unmarshal(message, &data)
			if err != nil {
				panic(err)
			}

			if data["type"] == "ping" {
				c.WriteJSON(map[string]interface{}{
					"type": "pong",
				})
			} else if data["type"] == "trade" {
				for _, trade := range data["data"].([]interface{}) {
					serialized, err := json.Marshal(trade)
					if err != nil {
						panic(err)
					}
					os.Stdout.Write(append(serialized, '\r', '\n'))
				}
			}
			if _, ok := data[""]; ok {
				continue
			}

		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 60)

		for _, symbol := range ctx.StringSlice("symbol") {
			c.WriteJSON(map[string]interface{}{
				"type":   "subscribe",
				"symbol": symbol,
			})

		}

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				c.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()

	<-done
	return nil
}

func main() {
	app := &cli.App{
		Name:   "binance-trade-tail",
		Usage:  "tail JSON crypto trade data from the binance streaming api",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "symbol",
				Usage: "a symbol to subscribe too",
			},
			&cli.StringFlag{
				Name:   "token",
				Usage:  "finnhub api token",
				EnvVar: "FINNHUB_TOKEN",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
