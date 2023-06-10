package experiments

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/joho/godotenv"
)

const envPrefix = "TASK_X_"

var Flags struct{}

func Parse() error {
	if err := readDotEnv(); err != nil {
		return err
	}
	t := reflect.TypeOf(&Flags)
	v := reflect.ValueOf(&Flags)
	if t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", v.Kind())
	}
	for i := 0; i < t.Elem().NumField(); i++ {
		fieldT := t.Elem().Field(i)
		fieldV := v.Elem().Field(i)
		if fieldT.Type.Kind() != reflect.Bool {
			return fmt.Errorf("task: expected bool, got %T", fieldV.Kind())
		}
		if !fieldV.CanSet() {
			return fmt.Errorf("task: cannot set experiment field: %q", fieldT.Name)
		}
		xName := fieldT.Tag.Get("x")
		xEnabled := parseEnv(xName)
		fieldV.SetBool(xEnabled)
	}
	return nil
}

func envName(xName string) string {
	xName = strings.ToUpper(xName)
	xName = strings.ReplaceAll(xName, " ", "_")
	xName = fmt.Sprintf("%s%s", envPrefix, xName)
	return xName
}

func parseEnv(xName string) bool {
	return os.Getenv(envName(xName)) == "1"
}

func readDotEnv() error {
	env, err := godotenv.Read()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	// If the env var is an experiment, set it.
	for key, value := range env {
		if strings.HasPrefix(key, envPrefix) {
			os.Setenv(key, value)
		}
	}
	return nil
}
