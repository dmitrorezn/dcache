package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/hashicorp/raft"
	"io"
	"strconv"
	"sync"
	"time"
)

type Cmd uint8

const (
	Undefined Cmd = iota
	Get
	Set
	Del
	Rename
	Wait
	Continue
	lastCmd
)

type Command struct {
	Cmd Cmd

	Payload []byte
	W       io.Writer
}

type Result struct {
}

type request struct {
	cmd    Command
	keys   []string
	value  []byte
	values [][]byte
	ack    chan error
}

type Storage struct {
	values map[string][]byte

	wg       sync.WaitGroup
	requests chan *request
	quit     chan struct{}
	pause    chan struct{}
	start    chan struct{}
}

type Cfg struct {
	Timeout time.Duration
}

type IStorage interface {
	Do(ctx context.Context, cmd Command) error
	Get(ctx context.Context, cmd Command) error
	Set(ctx context.Context, cmd Command) error
	Del(ctx context.Context, cmd Command) error
	Rename(ctx context.Context, cmd Command) error
	//Join(ctx context.Context, addr string) error
	CloseAndWait() error
}

var _ IStorage = new(Storage)
var _ IStorage = new(ReplicatorStorage)

type ReplicatorStorage struct {
	IStorage

	srvID raft.ServerID
	raft  *raft.Raft
	wg    sync.WaitGroup
}

func NewReplicator(s IStorage, srvID raft.ServerID, raft *raft.Raft) *ReplicatorStorage {
	rs := &ReplicatorStorage{
		IStorage: s,
		raft:     raft,
		srvID:    srvID,
	}

	return rs
}

func (r *ReplicatorStorage) CloseAndWait() error {
	defer r.wg.Wait()

	return r.IStorage.CloseAndWait()
}

func (r *ReplicatorStorage) apply(cmd Command) {
	buf := make([]byte, uint8ByteSize+len(cmd.Payload))
	n := binary.PutUvarint(buf[0:uint8ByteSize], uint64(cmd.Cmd))
	copy(buf[n:], cmd.Payload)
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		r.raft.Apply(buf, 5*time.Second)
	}()
}

func (r *ReplicatorStorage) Set(ctx context.Context, cmd Command) error {
	cmd.Cmd = Set
	r.apply(cmd)

	return r.IStorage.Set(ctx, cmd)
}

func (r *ReplicatorStorage) Del(ctx context.Context, cmd Command) error {
	r.wg.Add(1)
	cmd.Cmd = Del
	go func() {
		defer r.wg.Done()
		r.apply(cmd)
	}()

	return r.IStorage.Del(ctx, cmd)
}

func (r *ReplicatorStorage) Rename(ctx context.Context, cmd Command) error {
	r.wg.Add(1)
	cmd.Cmd = Rename
	go func() {
		defer r.wg.Done()
		r.apply(cmd)
	}()

	return r.IStorage.Rename(ctx, cmd)
}

func New() *Storage {
	s := &Storage{
		wg:       sync.WaitGroup{},
		values:   make(map[string][]byte),
		quit:     make(chan struct{}),
		pause:    make(chan struct{}),
		start:    make(chan struct{}),
		requests: make(chan *request, 10_000),
	}

	return s
}

func (s *Storage) Run(ctx context.Context) {
	defer s.wg.Wait()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.process(ctx)
	}()
}

func (s *Storage) Apply(log *raft.Log) interface{} {
	r, err := parseRPC(Command{
		Cmd:     Undefined,
		Payload: log.Data,
	})
	if err != nil {
		return err
	}
	select {
	case s.requests <- r:
	case <-s.quit:
		return ErrStorageClosed
	}

	return nil
}

func (s *Storage) Do(ctx context.Context, cmd Command) error {
	r, err := parseRPC(cmd)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.requests <- r:
	case <-s.quit:
		return ErrStorageClosed
	}

	return <-r.ack
}

var _ raft.FSMSnapshot = new(Storage)

func (s *Storage) Snapshot() (raft.FSMSnapshot, error) {
	s.requests <- &request{
		cmd: Command{
			Cmd: Undefined,
		},
	}
	return s, nil
}

func (s *Storage) Persist(sink raft.SnapshotSink) error {
	s.pause <- struct{}{}
	s.values["id"] = []byte(sink.ID())
	if err := json.NewEncoder(sink).Encode(s.values); err != nil {
		return errors.Join(
			sink.Cancel(),
			sink.Close(),
			err,
		)
	}

	return sink.Close()
}

func (s *Storage) Release() {
	s.start <- struct{}{}
}

func (s *Storage) Restore(snapshot io.ReadCloser) error {
	kv := make(map[string][]byte)
	if err := json.NewDecoder(snapshot).Decode(&kv); err != nil {
		return err
	}
	s.do(func() {
		keys := make([]string, 0, len(kv))
		values := make([][]byte, 0, len(kv))

		for k, v := range kv {
			keys = append(keys, k)
			values = append(values, v)
		}
		s.requests <- &request{
			cmd: Command{
				Cmd: Set,
			},
			keys:   keys,
			values: values,
			ack:    make(chan error),
		}
	})

	return snapshot.Close()
}

var ErrNIL = errors.New("nil")
var ErrWriteResult = errors.New("error write result")

func (s *Storage) do(fn func()) {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		fn()
	}()
}

func (r *Storage) CloseAndWait() error {
	close(r.quit)
	r.wg.Wait()

	return nil
}

func (s *Storage) process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.quit:
			return
		case <-s.pause:
			<-s.start
		case r := <-s.requests:
			switch r.cmd.Cmd {
			case Get:
				fmt.Println(s.values)
				res, ok := s.values[r.keys[0]]
				if !ok {
					r.ack <- ErrNIL
					close(r.ack)
					break
				}

				fmt.Println("res", string(res))
				s.do(func() {
					defer close(r.ack)
					fmt.Println("res", string(res))

					n, err := r.cmd.W.Write(res)
					if err != nil {
						r.ack <- err
						return
					}
					if n != len(res) {
						r.ack <- ErrWriteResult
					}
				})
			case Set:

				for i, k := range r.keys {
					s.values[k] = r.values[i]
				}
				fmt.Println(s.values)

				close(r.ack)

			case Del:
				for _, k := range r.keys {
					delete(s.values, k)
				}
				close(r.ack)

			case Rename:
				res, ok := s.values[r.keys[0]]
				if !ok {
					r.ack <- ErrNIL
					close(r.ack)
					break
				}
				delete(s.values, r.keys[0])
				s.values[r.keys[1]] = res

				close(r.ack)
			}
		}
	}
}

const (
	uint8ByteSize  = 1
	uint32ByteSize = 4
)

func readKey(cmd Command) ([]byte, int, error) {
	if len(cmd.Payload) < uint8ByteSize {
		return nil, 0, errors.New("wrong len" + fmt.Sprint(len(cmd.Payload)))
	}
	l, payload, ok := bytes.Cut(cmd.Payload, []byte{':'})
	if !ok {
		return nil, 0, errors.New("cut fail")
	}
	keyLen, err := strconv.Atoi(string(l))
	if err != nil {
		return nil, 0, err
	}
	if keyLen > len(payload) {
		return nil, 0, fmt.Errorf("key size is to big %d", keyLen)
	}
	return payload[:keyLen], int(keyLen), nil
}

func parseRPC(cmd Command) (*request, error) {
	if cmd.Cmd == Undefined {
		c, n := binary.Uvarint(cmd.Payload[:uint8ByteSize])
		if n != uint8ByteSize {
			return nil, fmt.Errorf("error parze cmd size %d", n)
		}
		cmd.Cmd = Cmd(c)
		cmd.Payload = cmd.Payload[:uint8ByteSize]
	}

	key, kl, err := readKey(cmd)
	if err != nil {
		return nil, fmt.Errorf("readKey %w", err)
	}
	var keys = []string{
		string(key),
	}

	switch cmd.Cmd {
	case Rename:
		k, n, err := readKey(cmd)
		if err != nil {
			return nil, fmt.Errorf("error parze cmd size %d", n)
		}
		keys = append(keys, string(k))
		kl += uint32ByteSize + n
	}

	return &request{
		cmd:    cmd,
		keys:   keys,
		values: [][]byte{cmd.Payload[uint32ByteSize+kl:]},
		ack:    make(chan error),
	}, nil
}

func (s *Storage) Get(ctx context.Context, cmd Command) error {
	cmd.Cmd = Get
	r, err := parseRPC(cmd)
	if err != nil {
		return err
	}
	fmt.Println("CMD", cmd)

	return s.applyRPC(ctx, r)
}

var ErrStorageClosed = errors.New("storage closed")

func (s *Storage) applyRPC(ctx context.Context, r *request) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.requests <- r:
	case <-s.quit:
		return ErrStorageClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-r.ack:
		return err
	}
}

func (s *Storage) Set(ctx context.Context, cmd Command) error {
	cmd.Cmd = Set
	r, err := parseRPC(cmd)
	if err != nil {
		return err
	}

	return s.applyRPC(ctx, r)
}

func (s *Storage) Del(ctx context.Context, cmd Command) error {
	cmd.Cmd = Del
	r, err := parseRPC(cmd)
	if err != nil {
		return err
	}

	return s.applyRPC(ctx, r)
}

func (s *Storage) Rename(ctx context.Context, cmd Command) error {
	cmd.Cmd = Rename
	r, err := parseRPC(cmd)
	if err != nil {
		return err
	}

	return s.applyRPC(ctx, r)
}

func (s *Storage) Join(ctx context.Context, cmd Command) error {
	fmt.Println("joined")

	return nil
}

type ActorStorage struct {
	IStorage

	pid      []*actor.PID
	engine   *actor.Engine
	commands chan Command
}

func NewActorStorage(s IStorage, commands chan Command, engine *actor.Engine) *ActorStorage {
	rs := &ActorStorage{
		IStorage: s,
		engine:   engine,
		commands: commands,
	}

	return rs
}

func (r *ActorStorage) apply(cmd Command) {
	select {
	case r.commands <- cmd:
	default:
		fmt.Println("ERROR apply CMD")
	}
}

func (r *ActorStorage) Set(ctx context.Context, cmd Command) error {
	cmd.Cmd = Set
	go r.apply(cmd)

	return r.IStorage.Set(ctx, cmd)
}

func (r *ActorStorage) Del(ctx context.Context, cmd Command) error {
	cmd.Cmd = Del

	go r.apply(cmd)

	return r.IStorage.Del(ctx, cmd)
}

func (r *ActorStorage) Rename(ctx context.Context, cmd Command) error {
	cmd.Cmd = Rename

	go r.apply(cmd)

	return r.IStorage.Rename(ctx, cmd)
}
