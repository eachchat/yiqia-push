package vivo

import (
	"context"
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type VIVO struct {
	endpoints *Endpoints
}

func NewPushClient(cfg *Config) (push.Push, error) {
	endpoints, err := newEndpoints(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed create VIVO endpoints: %v", err)
	}
	return &VIVO{
		endpoints: endpoints,
	}, nil
}

func (p *VIVO) PushNotice(ctx context.Context, message *push.Message) error {
	_, err := p.endpoints.PushNoticeEndpoint(ctx, message)
	return err
}

type Config struct {
	AppID     string `yaml:"app_id"`
	AppKey    string `yaml:"app_key"`
	AppSecret string `yaml:"app_secret"`
}

func (c *Config) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("app id is required")
	}
	if c.AppKey == "" {
		return fmt.Errorf("app key is required")
	}
	if c.AppSecret == "" {
		return fmt.Errorf("app secret is required")
	}
	return nil
}
