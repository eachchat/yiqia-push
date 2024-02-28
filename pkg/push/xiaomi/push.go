package xiaomi

import (
	"context"
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type XIAOMI struct {
	endpoints *Endpoints
}

func NewPushClient(cfg *Config) (push.Push, error) {
	endpoints, err := newEndpoints(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed create XIAOMI endpoints: %v", err)
	}
	return &XIAOMI{
		endpoints: endpoints,
	}, nil
}

func (p *XIAOMI) PushNotice(ctx context.Context, pushRequest *push.Message) error {
	_, err := p.endpoints.PushNoticeEndpoint(ctx, pushRequest)
	return err
}

type Config struct {
	AppPkgName string `yaml:"app_pkg_name"`
	AppSecret  string `yaml:"app_secret"`
	ChannelID  string `yaml:"channel_id"`
}

func (c *Config) Validate() error {
	if c.AppPkgName == "" {
		return fmt.Errorf("app pkg name is required")
	}
	if c.AppSecret == "" {
		return fmt.Errorf("app secret is required")
	}
	if c.ChannelID == "" {
		return fmt.Errorf("channel id is required")
	}
	return nil
}
