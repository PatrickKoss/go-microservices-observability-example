package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-microservices-observability/pkg/tracing"
	"net/http"
)

type Config struct {
	Address    string
	HTTPClient *http.Client
	Tracer     tracing.Tracer
}

type Client interface {
	Authenticate(ctx context.Context, user User) error
}

type client struct {
	config *Config
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c client) Authenticate(ctx context.Context, user User) error {
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/authenticate", c.config.Address),
		bytes.NewBuffer(b),
	)
	if err != nil {
		return err
	}

	c.config.Tracer.InjectHTTP(ctx, req.Header)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("authentication failed")
	}

	return nil
}

func NewClient(config *Config) Client {
	return &client{
		config: config,
	}
}
