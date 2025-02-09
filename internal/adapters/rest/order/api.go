package order

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	order_repository "go-microservices-observability/internal/adapters/repository/order"
	"go-microservices-observability/internal/adapters/user"
	"go-microservices-observability/internal/domain"
	"go-microservices-observability/internal/services/order"
	"go-microservices-observability/pkg/tracing"
	"net/http"
	"net/http/httptest"
	"strings"
)

type Server struct {
	e            *echo.Echo
	orderService order.Service
	userClient   user.Client
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

func NewServer(orderService order.Service, tracer tracing.Tracer, userClient user.Client) *Server {
	e := echo.New()

	s := &Server{
		e:            e,
		orderService: orderService,
		userClient:   userClient,
	}

	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(echo.WrapMiddleware(tracing.NewTracingMiddleware(tracer)))
	e.Use(BasicAuthMiddleware(userClient))

	e.GET("/orders", func(c echo.Context) error {
		orders, err := s.orderService.List(c.Request().Context())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, orders)
	})

	e.GET("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		order, err := s.orderService.Get(c.Request().Context(), id)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, order)
	})

	e.POST("/orders", func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return err
		}

		if err := s.orderService.Create(c.Request().Context(), &order); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, order)
	})

	e.PUT("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return err
		}
		order.ID = id
		if err := s.orderService.Update(c.Request().Context(), &order); err != nil {
			return err
		}

		return c.NoContent(http.StatusNoContent)
	})

	e.DELETE("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		if err := s.orderService.Delete(c.Request().Context(), id); err != nil {
			return err
		}

		return c.NoContent(http.StatusNoContent)
	})

	return s
}

func customHTTPErrorHandler(rootError error, c echo.Context) {
	println("customHTTPErrorHandler: ", rootError.Error())
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

	if errors.Is(err, order_repository.ErrOrderAlreadyExists) {
		return ctx.JSON(http.StatusConflict, ErrorMessageResp{
			Message: "order already exists",
		})
	}

	if errors.Is(err, order_repository.ErrOrderNotFound) {
		return ctx.JSON(http.StatusNotFound, ErrorMessageResp{
			Message: "order not found",
		})
	}

	return findHTTPError(ctx, errors.Unwrap(err))
}

type ErrorMessageResp struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func BasicAuthMiddleware(userClient user.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			authParts := strings.SplitN(authHeader, " ", 2)
			if len(authParts) != 2 || authParts[0] != "Basic" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
			}

			payload, err := base64.StdEncoding.DecodeString(authParts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid base64 encoding")
			}

			credentials := strings.SplitN(string(payload), ":", 2)
			if len(credentials) != 2 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization value")
			}

			user := user.User{
				Username: credentials[0],
				Password: credentials[1],
			}

			if err := userClient.Authenticate(c.Request().Context(), user); err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication failed")
			}

			return next(c)
		}
	}
}
