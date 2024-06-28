package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	"github.com/dmitrorezn/depless/internal/auth"
)

func newErr(code int, err error) error {
	return &echo.HTTPError{
		Code:     code,
		Message:  err,
		Internal: err,
	}
}

type HTMLErr struct {
	echo.HTTPError
}

func (e HTMLErr) Message() string {
	return fmt.Sprint(e.HTTPError.Message)
}

func (e HTMLErr) Code() string {
	return fmt.Sprint(e.HTTPError.Code)
}

func newHTMLErr(code int, err error, msg any) error {
	return &echo.HTTPError{
		Code:     code,
		Message:  msg,
		Internal: err,
	}
}

func newRedirectErr(err error, dst string, code ...int) error {
	return &RedirectErr{
		Err:         err,
		Destination: dst,
		Code:        append(code, http.StatusFound)[0],
	}
}

type RedirectErr struct {
	Destination string
	Err         error
	Code        int
}

func (re RedirectErr) Error() string {
	return re.Err.Error()
}

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		if he.Internal != nil {
			var herr *echo.HTTPError
			if errors.As(he.Internal, &herr) {
				he = herr
			}
		}
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}
	}
	// Issue #1426
	code := he.Code
	message := he.Message

	switch m := he.Message.(type) {
	case string:
		if c.Echo().Debug {
			message = echo.Map{"message": m, "error": err.Error()}
		} else {
			message = echo.Map{"message": m}
		}
	case json.Marshaler:
		// do nothing - this type knows how to format itself to JSON
	case RedirectErr:
		_ = c.Redirect(http.StatusFound, m.Destination)
		c.Echo().Logger.Error(err)
		return
	case *RedirectErr:
		_ = c.Redirect(http.StatusFound, m.Destination)
		c.Echo().Logger.Error(err)
		return
	case auth.ErrAuth:
		message = echo.Map{"message": m.Error()}
	case error:
		message = echo.Map{"message": m.Error()}
	case HTMLErr:
		//_ = render(c, errtempl.Err(&m))

		return
	}
	// Send response
	if c.Request().Method == http.MethodHead { // Issue #608
		err = c.NoContent(he.Code)
	} else {
		if errors.Is(err, sql.ErrNoRows) {
			//code = http.StatusNotFound todo decide needed of not
		}
		if errors.As(err, &auth.ErrAuth{}) {
			code = http.StatusUnauthorized
		}
		err = errors.Join(
			err,
			c.JSON(code, message),
		)
	}
	if err != nil {
		c.Echo().Logger.Error(err)
	}
}
