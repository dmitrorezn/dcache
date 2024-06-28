package main

import (
	"context"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/dmitrorezn/dcache/storage"
	"log"
)

type Server struct {
	pids     map[*actor.PID]struct{}
	commands chan storage.Command
	store    storage.IStorage
	cancel   context.CancelFunc
	cluster  *cluster.Cluster
}

func NewServer(store storage.IStorage, cluster *cluster.Cluster, commands chan storage.Command) actor.Producer {
	return func() actor.Receiver {
		return &Server{
			cluster:  cluster,
			commands: commands,
			store:    store,
			pids:     make(map[*actor.PID]struct{}),
		}
	}
}

type ReplicateCommand struct {
	Cmd     int
	Payload []byte
}

func (s *Server) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:

	case *cluster.Activation, cluster.ActivationEvent:
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		for i := 0; i < 10; i++ {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case cmd := <-s.commands:
						for pid := range s.pids {
							log.Println("REPLICATE TO", pid.Address, string(cmd.Payload))
							s.cluster.Engine().Send(pid, ReplicateCommand{
								Cmd:     int(cmd.Cmd),
								Payload: cmd.Payload,
							})
						}
					}
				}
			}()
		}
	case cluster.MemberJoinEvent:
		workerID := s.cluster.Activate("worker", cluster.NewActivationConfig().
			WithID(*nodeID),
		)
		fmt.Println("MemberJoinEvent workerID", workerID, msg)

		s.pids[c.Sender()] = struct{}{}
		fmt.Println("PIDS", s.pids)
		c.Send(workerID, Connect{})
	case actor.Stopped:
		s.cancel()
		if err := s.store.CloseAndWait(); err != nil {
			fmt.Println("CloseAndWait", err)
		}
	case ReplicateCommand:
		if c.Sender().GetAddress() == s.cluster.PID().GetAddress() {
			fmt.Println("LOCAL")
		} else {
			s.replicate(c.Context(), msg)
		}
	case Connect:
		fmt.Println("Connect", msg, "id", c.Sender().ID)

	case Disconnect:
		fmt.Println("Disconnect", msg)
	}
}

func (s *Server) replicate(ctx context.Context, msg ReplicateCommand) {
	fmt.Println("Replicate Command Receive", msg)
	var err error
	cmd := storage.Cmd(msg.Cmd)
	command := storage.Command{
		Cmd:     cmd,
		Payload: msg.Payload,
	}
	switch cmd {
	case storage.Set:
		err = s.store.Set(ctx, command)
	case storage.Del:
		err = s.store.Del(ctx, command)
	case storage.Rename:
		err = s.store.Rename(ctx, command)
	}
	_ = err
}

type Connect struct {
}
type Replicate struct{}
type Disconnect struct{}
