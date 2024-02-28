package config

import (
	"fmt"
	"reflect"
)

type Config interface {
	Validate() error
}

// validateConfig validates the config
func ValidateConfig[T any](cfg T) error {
	return rangeField(cfg, func(c Config) error {
		if err := c.Validate(); err != nil {
			return err
		}
		return nil
	})
}

// rangeField iterates over the fields of a struct and calls the given function
func rangeField[T any](ptr any, fn func(T) error) error {
	v := reflect.ValueOf(ptr).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.IsNil() && f.CanSet() {
			iface := f.Interface()
			if opts, ok := iface.(T); ok {
				if err := fn(opts); err != nil {
					return fmt.Errorf("failed validate config: %v, key: %s", err, v.Type().Field(i).Name)
				}
			}
		}
	}
	return nil
}
