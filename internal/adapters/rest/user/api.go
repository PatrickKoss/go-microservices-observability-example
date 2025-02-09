package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go-microservices-observability/internal/services/order"
	"go-microservices-observability/pkg/tracing"
	"net/http"
	"net/http/httptest"
)

type Server struct {
	e            *echo.Echo
	orderService order.Service
}

func (s *Server) ListenAndServe(port int) error {
	err := s.e.Start(fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}

func (s *Server) Test(req *http.Request) *http.Response {
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)

	return rec.Result()
}

func NewServer(tracer tracing.Tracer) *Server {
	e := echo.New()

	s := &Server{
		e: e,
	}

	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(echo.WrapMiddleware(tracing.NewTracingMiddleware(tracer)))

	e.POST("/authenticate", func(c echo.Context) error {
		return c.JSON(http.StatusOK, ErrorMessageResp{
			Message: "authenticated",
		})
	})

	return s
}

func customHTTPErrorHandler(rootError error, c echo.Context) {
	err := findHTTPError(c, rootError)

	if err == nil {
		err = rootError
	}

	c.Echo().DefaultHTTPErrorHandler(err, c)
}

func findHTTPError(ctx echo.Context, err error) error {
	if err == nil {
		return nil
	}

	var e *echo.HTTPError
	if errors.As(err, &e) {
		return e
	}

	return findHTTPError(ctx, errors.Unwrap(err))
}

type ErrorMessageResp struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}
