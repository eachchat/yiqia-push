package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/eachchat/yiqia-push/pkg/log"
	"github.com/eachchat/yiqia-push/pkg/notify"
	"github.com/eachchat/yiqia-push/pkg/push/overall"
	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v3"
)

// config is the configuration of the server.
type config struct {
	// Addr is the address to listen on.
	// Default: :80
	Addr string `yaml:"addr"`

	// LogLevel is the log level.
	// Default: info
	LogLevel string         `yaml:"log_level"`
	Pusher   overall.Config `yaml:"pusher"`
	Notify   notify.Config  `yaml:"notify"`
}

func (c *config) Validate() error {
	if c.Addr == "" {
		c.Addr = ":80"
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	err := c.Notify.Validate()
	if err != nil {
		return fmt.Errorf("invalid notify config: %v", err)
	}

	err = c.Pusher.Validate()
	if err != nil {
		return fmt.Errorf("invalid pusher config: %v", err)
	}

	return nil
}

// getConfig reads the configuration from the given file.
func getConfig(path string) (*config, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fail open config file: %v", err)
	}
	defer fd.Close()

	conf := new(config)
	err = yaml.NewDecoder(fd).Decode(conf)
	if err != nil {
		return nil, fmt.Errorf("fail decode config file: %v", err)
	}

	err = conf.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return conf, nil
}

// Main starts the server and returns a function to stop it.
func Main(ctx context.Context) func() {
	var configPath string
	flag.StringVar(&configPath, "c", "./config.yaml", "config path. default: ./config.yaml")
	flag.Parse()

	cfg, err := getConfig(configPath)
	if err != nil {
		stdlog.Fatalf("fail get config: %v", err)
	}

	logger := log.NewLogger(cfg.LogLevel)

	overAll, err := overall.New(&cfg.Pusher)
	if err != nil {
		level.Error(logger).Log("msg", "fail new overall", "err", err)
		os.Exit(1)
	}

	s := http.Server{
		Addr:    cfg.Addr,
		Handler: notify.New(ctx, &cfg.Notify, overAll, logger),
	}

	level.Info(logger).Log("msg", "start server", "addr", cfg.Addr)
	go func() {
		err = s.ListenAndServe()
		if err != nil {
			level.Error(logger).Log("msg", "fail listen and serve", "err", err)
			os.Exit(1)
		}
	}()

	return func() {
		_ = s.Shutdown(ctx)
	}
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)

	stop := Main(ctx)

	<-ctx.Done()
	cancel()
	stop()

	<-time.After(5 * time.Second)
	fmt.Println("shutdown gracefully")
}
