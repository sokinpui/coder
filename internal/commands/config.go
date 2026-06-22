package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
	"gopkg.in/yaml.v3"
	"reflect"
	"strconv"
	"strings"
)

func init() {
	registerCommand("config", configCmd, "show/update/remove local config", configArgumentCompleter)
}

func configArgumentCompleter(cfg *config.Config, prefix string) []string {
	return collectConfigKeys(reflect.ValueOf(cfg), "")
}

func configCmd(args string, s SessionController) (CommandOutput, bool) {
	fields := strings.Fields(args)

	if len(fields) == 0 {
		return displayFullConfig(s)
	}

	para := fields[0]
	var val any = nil
	action := "Removed"

	if len(fields) > 1 {
		rawVal := strings.Join(fields[1:], " ")
		val = parseConfigValue(rawVal)
		action = "Updated"
	}

	if err := config.UpdateLocalConfig(para, val); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error updating local config: %v", err)}, false
	}

	if err := s.ReloadConfig(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Config saved to file, but failed to reload session: %v", err)}, false
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("%s local configuration: %s", action, para)}, true
}

func displayFullConfig(s SessionController) (CommandOutput, bool) {
	data, err := yaml.Marshal(s.GetConfig())
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error marshaling config: %v", err)}, false
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: string(data)}, true
}

func parseConfigValue(v string) any {
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

func collectConfigKeys(v reflect.Value, prefix string) []string {
	var keys []string
	t := v.Type()

	if t.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		t = v.Type()
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			tag := field.Tag.Get("mapstructure")
			if tag == "-" || (tag == "" && field.Tag.Get("yaml") == "-") {
				continue
			}

			name := strings.Split(tag, ",")[0]
			if name == "" {
				name = strings.ToLower(field.Name)
			}

			fullKey := name
			if prefix != "" {
				fullKey = prefix + "." + name
			}

			subKeys := collectConfigKeys(v.Field(i), fullKey)
			if len(subKeys) == 0 {
				keys = append(keys, fullKey)
			} else {
				keys = append(keys, subKeys...)
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			name := k.String()
			fullKey := prefix + "." + name
			subKeys := collectConfigKeys(v.MapIndex(k), fullKey)
			keys = append(keys, subKeys...)
		}
	}

	return keys
}
