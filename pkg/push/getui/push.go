package getui

import (
	"context"
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type GETUI struct {
	endpoints *Endpoints
}

func New(conf *Config) (push.Push, error) {
	endpoints, err := newEndpoints(context.Background(), conf)
	if err != nil {
		return nil, fmt.Errorf("failed create GETUI endpoints: %v", err)
	}
	return &GETUI{
		endpoints: endpoints,
	}, nil
}

func (p *GETUI) PushNotice(ctx context.Context, message *push.Message) error {
	_, err := p.endpoints.PushNoticeEndpoint(ctx, message)
	return err
}

type Config struct {
	AppID        string `yaml:"app_id"`
	AppKey       string `yaml:"app_key"`
	MasterSecret string `yaml:"master_secret"`
}

func (c *Config) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("app id is required")
	}
	if c.AppKey == "" {
		return fmt.Errorf("app key is required")
	}
	if c.MasterSecret == "" {
		return fmt.Errorf("master secret is required")
	}
	return nil
}
