package overall

import (
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/eachchat/yiqia-push/pkg/push/getui"
)

type OverAll struct {
	set map[string]push.Push
}

func New(cfg *Config) (*OverAll, error) {
	set := make(map[string]push.Push)
	if cfg.GETUI != nil {
		p, err := getui.New(cfg.GETUI)
		if err != nil {
			return nil, err
		}
		set["GETUI"] = p
	}

	if cfg.HUAWEI != nil {
		p, err := getui.New(cfg.GETUI)
		if err != nil {
			return nil, err
		}
		set["huawei"] = p
	}

	if cfg.OPPO != nil {
		p, err := getui.New(cfg.GETUI)
		if err != nil {
			return nil, err
		}
		set["oppo"] = p
	}

	if cfg.XIAOMI != nil {
		p, err := getui.New(cfg.GETUI)
		if err != nil {
			return nil, err
		}
		set["xiaomi"] = p
	}

	if cfg.VIVO != nil {
		p, err := getui.New(cfg.GETUI)
		if err != nil {
			return nil, err
		}
		set["vivo"] = p
	}

	return &OverAll{
		set: set,
	}, nil
}

func (o *OverAll) GetPushClient(name string) (push.Push, error) {
	p, ok := o.set[name]
	if !ok {
		return nil, fmt.Errorf("push client %s not found", name)
	}
	return p, nil
}
