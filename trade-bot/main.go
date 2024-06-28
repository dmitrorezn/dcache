package main

import (
	"context"
	"fmt"
	"github.com/binance/binance-connector-go"
)

func main() {

	appCfg := GetCfg()

	client:=newClient(appCfg)

	tradeclient := client.NewAccountApiTradingStatusService()

	resp,err := tradeclient.Do(context.Background())
	if err!=nil{
	fmt.Println("Do",err)
	return
	}

	fmt.Println("Do resp",resp)

}

func newClient(cfg *Config) *binance_connector.Client {
	return binance_connector.NewClient(cfg.ApiKey, cfg.SecretKey)
}