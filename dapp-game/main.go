package main

import (
	"context"
	"encoding/hex"
	"fmt"
	httpserver "github.com/dmitrorezn/dapp_game/pkg/server/http"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	game "github.com/dmitrorezn/dapp_game/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	g, ctx := errgroup.WithContext(context.Background())
	defer g.Wait()

	ctx, cancel := signal.NotifyContext(ctx, os.Kill, os.Interrupt)
	defer cancel()

	engine := NewEngine()
	setups := []func(ctx context.Context) error{
		engine.SetupContract,
		engine.SetupGame,
		engine.SetupServer,
	}
	for _, s := range setups {
		if err := s(ctx); err != nil {
			log.Fatalln(err)
		}
	}

	runups := []func(ctx context.Context) error{
		engine.SetupServer,
		engine.RegisrerWorkers,
		engine.Close,
	}
	for _, run := range runups {
		g.Go(func() error {
			return run(ctx)
		})
	}
}

type Engine struct {
	gameAddr common.Address
	game     *game.Game
	client   *ethclient.Client
	subs     []event.Subscription
	wg       errgroup.Group
}

func NewEngine() *Engine {
	return &Engine{}
}

func nonce() *big.Int {
	return big.NewInt(rand.Int63())
}

var (
	privateKey = []byte("bb3b487e5fe2716e6b1b09840e41ba4be349b072ea022ffbf783f249ff3f563bb9664085699031482dba2ceb7d89b8eba1b1914c3b93632acf2f9856bd2ff2af")
)

var (
	src    = rand.NewSource(time.Now().UnixNano())
	rander = rand.New(src)
)

func randStringN(n int) string {
	r := make([]byte, n)
	_, _ = rander.Read(r)

	return hex.EncodeToString(r)
}

const (
	endpoint = "https://eth-mainnet.g.alchemy.com/v2/EwegiN7Inwcd2gHGr7LZCLqiMAk3Fqnb"
	baseAddr = "0xcA5e1C4D9E50134c87e64102bD875eB09268Cd48"
)

func (e *Engine) SetupContract(ctx context.Context) error {
	//transport, err := rpc.DialHTTP(endpoint)
	//if err != nil {
	//	return errors.Wrap(err, "Dial")
	//}
	addr := common.HexToAddress(baseAddr)

	privKey, err := crypto.GenerateKey()
	if err != nil {
		return errors.Wrap(err, "Dial")
	}
	_ = privKey
	alloc := core.GenesisAlloc{
		addr: core.GenesisAccount{
			Code: []byte(""),
			//PrivateKey: crypto.p.E(privKey.D),
			Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9)),
			Nonce:   nonce().Uint64(),
			Storage: map[common.Hash]common.Hash{},
		},
	}
	fmt.Println("addr", addr)
	backend := simulated.NewBackend(
		alloc,
		//simulated.WithBlockGasLimit(5),
		//simulated.WithCallGasLimit(5),
		//simulated.WithCallGasLimit(5),
		func(nodeConf *node.Config, ethConf *ethconfig.Config) {
			nodeConf.AllowUnprotectedTxs = true
		},
	)
	hash := backend.Commit()
	defer func() {
		if err == nil {
			err = backend.Close()
			return
		}
		backend.Rollback()
		err = fmt.Errorf("%w %w", err, backend.Close())
	}()
	fmt.Println("hash", hash)

	client := backend.Client()
	block, err := client.BlockByHash(ctx, hash)
	if err != nil {
		return errors.Wrap(err, "BlockByHash")
	}
	balance, err := client.BalanceAt(ctx, addr, block.Number())
	if err != nil {
		return errors.Wrap(err, "BalanceAt")
	}
	fmt.Println("balance", balance)

	//chainID, err := client.ChainID(ctx)
	//if err != nil {
	//	return errors.Wrap(err, "ChainID")
	//}
	sig := new(types.HomesteadSigner)

	txPpt := &bind.TransactOpts{
		From:  addr,
		Nonce: nonce(),
		Value: big.NewInt(1000),
		Signer: func(address common.Address, tx *types.Transaction) (_ *types.Transaction, err error) {
			//return tx, nil
			//ltx := &types.LegacyTx{
			//	Nonce:    tx.Nonce(),
			//	GasPrice: tx.GasPrice(),
			//	//To:       &address,
			//	Value: tx.Value(),
			//	Data:  tx.Data(),
			//	Gas:   tx.Gas(),
			//}
			//tx = types.NewTx(ltx)
			sign, err := crypto.Sign(tx.Hash().Bytes(), privKey)
			if err != nil {
				return nil, err
			}
			tx, err = tx.WithSignature(sig, sign)
			if err != nil {
				fmt.Println("WithSignature transaction", tx, "addr", address, err)
				return nil, err
			}
			fmt.Println("addr", address)

			return tx, nil
		},
		Context: ctx,
	}
	txPpt.GasPrice, err = client.SuggestGasPrice(ctx)
	if err != nil {
		return errors.Wrap(err, "EstimateGas")
	}
	txPpt.GasLimit = 100_000
	fmt.Println("txPpt.GasPrice", txPpt.GasLimit, txPpt.GasPrice)
	contractAddr, tx, dgame, err := game.DeployGame(txPpt, client)
	if err != nil {
		return errors.Wrap(err, "DeployGame")
	}
	e.gameAddr = contractAddr
	e.game = dgame

	fmt.Println("contractAddr", contractAddr, tx, dgame)

	return nil
}
func (e *Engine) SetupGame(ctx context.Context) error {
	version, err := e.game.Version(&bind.CallOpts{})
	if err != nil {
		return errors.Wrap(err, "Version")
	}

	fmt.Println(version, version)
	return nil
}

func (e *Engine) SetupServer(ctx context.Context) error {
	server := httpserver.New()
	server.Register("/bet", e.MakeBetHandler)
	server.Register("/win", e.WinHandler)
	server.Register("/balance", e.Balance)

	return server.Run(":10000")
}

func (e *Engine) RegisrerWorkers(ctx context.Context) error {
	var bets = make(chan *game.GameBetDone)
	sub, err := e.game.WatchBetDone(&bind.WatchOpts{Context: ctx}, bets)
	if err != nil {
		return err
	}
	e.subs = append(e.subs, sub)

	var wins = make(chan *game.GameWon)
	if sub, err = e.game.WatchWon(&bind.WatchOpts{Context: ctx}, wins); err != nil {
		return err
	}
	e.subs = append(e.subs, sub)

	e.wg.Go(func() error {
		return e.runWorkBets(ctx, bets)
	})
	e.wg.Go(func() error {
		return e.runWorkWins(ctx, wins)
	})

	return nil
}

func (e *Engine) runWorkBets(ctx context.Context, bets chan *game.GameBetDone) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case bet := <-bets:
			fmt.Println("bets", bet)
		}
	}
}

func (e *Engine) runWorkWins(ctx context.Context, wins chan *game.GameWon) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case win := <-wins:
			fmt.Println("wins", win)
		}
	}
}

type MakeBet struct {
	Addr   string
	Amount amount
}

type amount int64

func (a amount) toBig() *big.Int {
	return big.NewInt(int64(a / 100))
}

func (e *Engine) MakeBetHandler(c echo.Context) error {
	var request MakeBet
	if err := c.Bind(&request); err != nil {
		return err
	}

	addr := common.HexToAddress(request.Addr)

	txOpt := &bind.TransactOpts{
		From:  addr,
		Nonce: nonce(),
	}

	tx, err := e.game.MakeBet(txOpt, addr, request.Amount.toBig())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"hash": tx.Hash(),
		"data": string(tx.Data()),
		"cost": tx.Cost().Int64(),
	})
}

type WinRequest struct {
	Addr   string
	Amount amount
}

func (e *Engine) WinHandler(c echo.Context) error {
	var request WinRequest
	if err := c.Bind(&request); err != nil {
		return err
	}

	addr := common.HexToAddress(request.Addr)

	txOpt := &bind.TransactOpts{
		From:  addr,
		Nonce: nonce(),
	}

	tx, err := e.game.Win(txOpt, addr, request.Amount.toBig())
	if err != nil {
		return err
	}
	_ = tx

	return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
}
func (e *Engine) Balance(c echo.Context) error {
	var request WinRequest
	if err := c.Bind(&request); err != nil {
		return err
	}

	addr := common.HexToAddress(request.Addr)

	balance, err := e.game.GetBalance(&bind.CallOpts{
		From: e.gameAddr,
	}, addr)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"balance": amount(balance.Int64()) * 100,
	})
}

func (e *Engine) Close(ctx context.Context) error {
	<-ctx.Done()

	for _, sub := range e.subs {
		sub.Unsubscribe()
		if err := sub.Err(); err != nil {
			fmt.Println("Engine Close Unsubscribe", err)
		}
	}
	fmt.Println("Engine Closed")

	return nil
}
