package overall

import (
	"fmt"

	"github.com/eachchat/yiqia-push/pkg/config"
	"github.com/eachchat/yiqia-push/pkg/push/getui"
	"github.com/eachchat/yiqia-push/pkg/push/huawei"
	"github.com/eachchat/yiqia-push/pkg/push/oppo"
	"github.com/eachchat/yiqia-push/pkg/push/vivo"
	"github.com/eachchat/yiqia-push/pkg/push/xiaomi"
)

type Config struct {
	GETUI  *getui.Config  `json:"getui"`
	HUAWEI *huawei.Config `json:"huawei"`
	OPPO   *oppo.Config   `json:"oppo"`
	XIAOMI *xiaomi.Config `json:"xiaomi"`
	VIVO   *vivo.Config   `json:"vivo"`
}

// Validate validates the push config
func (c *Config) Validate() error {
	err := config.ValidateConfig[config.Config](c)
	if err != nil {
		return fmt.Errorf("failed validate push config: %v", err)
	}
	return nil
}
