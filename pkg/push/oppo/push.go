package oppo

import (
	"context"
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type OPPO struct {
	endpoints *Endpoints
}

func New(conf *Config) (push.Push, error) {
	endpoints, err := newEndpoints(context.Background(), conf)
	if err != nil {
		return nil, fmt.Errorf("failed create OPPO endpoints: %v", err)
	}
	return &OPPO{
		endpoints: endpoints,
	}, nil
}

func (p *OPPO) PushNotice(ctx context.Context, message *push.Message) error {
	_, err := p.endpoints.PushNoticeEndpoint(ctx, message)
	return err
}

type Config struct {
	AppKey       string `yaml:"app_key"`
	MasterSecret string `yaml:"master_secret"`
	ChannelID    string `yaml:"channel_id"`
}

func (c *Config) Validate() error {
	if c.AppKey == "" {
		return fmt.Errorf("app key is required")
	}
	if c.MasterSecret == "" {
		return fmt.Errorf("master secret is required")
	}
	if c.ChannelID == "" {
		return fmt.Errorf("channel id is required")
	}
	return nil
}
