package huawei

import (
	"context"
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type HUAWEI struct {
	endpoints *Endpoints
}

func New(cfg *Config) (push.Push, error) {
	endpoints, err := newEndpoints(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed create HUAWEI endpoints: %v", err)
	}
	return &HUAWEI{
		endpoints: endpoints,
	}, nil
}

func (p *HUAWEI) PushNotice(ctx context.Context, message *push.Message) error {
	_, err := p.endpoints.PushNoticeEndpoint(ctx, message)
	return err
}

type Config struct {
	ClientId       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
	TargetUserType int    `yaml:"target_user_type"`
}

func (c *Config) Validate() error {
	if c.ClientId == "" {
		return fmt.Errorf("client id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}
	return nil
}
