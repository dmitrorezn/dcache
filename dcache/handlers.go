package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dmitrorezn/dcache/storage"
)

type Cmd struct {
	Cmd     int    `json:"cmd"`
	Payload string `json:"payload"`
}

func ParseCmd(rc io.ReadCloser) (cmd Cmd, err error) {
	return cmd, errors.Join(
		json.NewDecoder(rc).Decode(&cmd),
		rc.Close(),
	)
}

func handleGet(s storage.IStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		cmd, err := ParseCmd(r.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)

			return
		}

		fmt.Println("cmd", cmd)
		if err = s.Get(r.Context(), storage.Command{
			Cmd:     storage.Get,
			Payload: []byte(cmd.Payload),
			W:       rw,
		}); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)

			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
func handleSet(s storage.IStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		cmd, err := ParseCmd(r.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		if err = s.Set(r.Context(), storage.Command{
			Cmd:     storage.Set,
			Payload: []byte(cmd.Payload),
			W:       rw,
		}); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
func handleDel(s storage.IStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		cmd, err := ParseCmd(r.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if err = s.Del(r.Context(), storage.Command{
			Cmd:     storage.Del,
			Payload: []byte(cmd.Payload),
			W:       rw,
		}); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func handleRename(s storage.IStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		cmd, err := ParseCmd(r.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if err = s.Rename(r.Context(), storage.Command{
			Cmd:     storage.Rename,
			Payload: []byte(cmd.Payload),
			W:       rw,
		}); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
