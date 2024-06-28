package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/anthdm/hollywood/cluster"
	"github.com/caarlos0/env"

	"github.com/dmitrorezn/dcache/server"
	"github.com/dmitrorezn/dcache/storage"
)

type Cfg struct {
	Port        string `env:"PORT" envDefault:"8080"`
	LeaderAddr  string `env:"LEADER_ADDR"`
	ClusterAddr string `env:"CLUSTER_ADDR"`
	//RaftAddr    string        `env:"RAFT_ADDR"`
	IsLeader bool          `env:"IS_LEADER"`
	Timeout  time.Duration `env:"TIMEOUT" envDefault:"15s"`
}

const (
	localhost = "127.0.0.1"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	var cfg Cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Parse", err)
	}

	fmt.Println("CFG", cfg)

	clusterAddr := cfg.ClusterAddr
	clusterCfg := cluster.NewConfig().
		WithID(*nodeID).
		WithListenAddr(clusterAddr).
		WithRegion("eu-west")

	clusterActor, err := cluster.New(clusterCfg)
	if err != nil {
		log.Fatal("cluster.New", err)
	}
	var (
		addr                = net.JoinHostPort(localhost, cfg.Port)
		srv                 = server.NewHTTP(addr)
		localStore          = storage.New()
		replicationCommands = make(chan storage.Command, 1024)
		producer            = NewServer(localStore, clusterActor, replicationCommands)
		srvPID              = clusterActor.Spawn(producer, "server-"+*nodeID)
		actorStorage        = storage.NewActorStorage(localStore, replicationCommands, clusterActor.Engine())
	)
	clusterActor.Engine().
		Subscribe(srvPID)

	log.Println("srvPID", srvPID)

	clusterActor.RegisterKind(
		"worker",
		NewServer(localStore, clusterActor, replicationCommands),
		cluster.NewKindConfig(),
	)

	log.Println("cluster.START", err)
	clusterActor.Start()
	log.Println("cluster.STARTED", err)

	mux := http.NewServeMux()
	mux.Handle("POST /get", handleGet(actorStorage))
	mux.Handle("POST /set", handleSet(actorStorage))
	mux.Handle("POST /del", handleDel(actorStorage))
	mux.Handle("POST /rename", handleRename(actorStorage))

	srv.Register(mux)

	wg := errgroup.Group{}
	wg.Go(func() error {
		localStore.Run(ctx)
		return nil
	})
	wg.Go(srv.Run)

	var shutdowns = []func() error{
		srv.Close,
		func() error {
			defer clusterActor.Stop()
			fmt.Println("STOPPING CLUSTER")
			return nil
		},
	}
	wg.Go(func() (err error) {
		<-ctx.Done()
		for _, sh := range shutdowns {
			err = errors.Join(err, sh())
		}
		return err
	})

	if err = wg.Wait(); err != nil {
		log.Fatal(err)
	}
}
