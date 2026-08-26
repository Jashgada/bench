package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type bodyProperty struct {
	name, kind string
	required   bool
}

func bodyProperties(schema json.RawMessage) []bodyProperty {
	var value struct {
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if json.Unmarshal(schema, &value) != nil {
		return nil
	}
	required := make(map[string]bool, len(value.Required))
	for _, name := range value.Required {
		required[name] = true
	}
	properties := make([]bodyProperty, 0, len(value.Properties))
	for name, property := range value.Properties {
		kind := property.Type
		if kind == "" || (kind != "string" && kind != "number" && kind != "integer" && kind != "boolean") {
			kind = "json"
		}
		properties = append(properties, bodyProperty{name: name, kind: kind, required: required[name]})
	}
	sort.Slice(properties, func(i, j int) bool { return properties[i].name < properties[j].name })
	return properties
}

func encodeBodyValue(value, kind string) (json.RawMessage, error) {
	if kind == "json" {
		if !json.Valid([]byte(value)) {
			return nil, fmt.Errorf("expected valid JSON")
		}
		return json.RawMessage(value), nil
	}
	if kind == "string" {
		return json.Marshal(value)
	}
	if kind == "boolean" {
		if value != "true" && value != "false" {
			return nil, fmt.Errorf("expected boolean")
		}
		return json.RawMessage(value), nil
	}
	if kind == "integer" {
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return nil, fmt.Errorf("expected integer")
		}
	}
	if kind == "number" {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return nil, fmt.Errorf("expected number")
		}
	}
	return json.RawMessage(strings.TrimSpace(value)), nil
}
