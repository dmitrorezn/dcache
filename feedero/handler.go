package main

import (
	"context"
	"github.com/labstack/echo"
	"net/http"
)

type (
	Command[C any]  func(ctx context.Context, cmd C) ([]interface{}, error)
	Query[Q, R any] func(ctx context.Context, query Q) (R, error)
)

type QueryHandler[Q, R any] struct {
	Query Query[Q, R]
}

func NewQueryHandler[Q, R any](query Query[Q, R]) *QueryHandler[Q, R] {
	return &QueryHandler[Q, R]{
		Query: query,
	}
}

type Handler = echo.HandlerFunc

func (h *QueryHandler[Q, R]) POST() Handler {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var query Q
		if err := c.Bind(&query); err != nil {
			return err
		}
		result, err := h.Query(ctx, query)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}

type QueryWithParam interface {
	WithParam(name, value string) error
}

func (h *QueryHandler[Q, R]) GET(param ...string) Handler {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var query Q
		if len(param) > 0 {
			switch q := any(query).(type) {
			case QueryWithParam:
				if err := q.WithParam(param[0], c.Param(param[0])); err != nil {
					return err
				}
			}
		}
		result, err := h.Query(ctx, query)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}

type CommandHandler[C any] struct {
	Command Command[C]
}

func NewCommandHandler[C any](command Command[C]) *CommandHandler[C] {
	return &CommandHandler[C]{
		Command: command,
	}
}

func (h *CommandHandler[C]) POST() Handler {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var command C
		if err := c.Bind(&command); err != nil {
			return err
		}
		_, err := h.Command(ctx, command)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{})
	}
}
