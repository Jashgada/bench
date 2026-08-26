package cmd

import (
	"sort"
	"strings"
)

func firstTag(api API) string {
	if len(api.Tags) == 0 {
		return ""
	}
	return api.Tags[0]
}

func compareAPIs(a, b API, key string) int {
	switch key {
	case "method":
		if c := strings.Compare(strings.ToUpper(a.Method), strings.ToUpper(b.Method)); c != 0 {
			return c
		}
	case "tag":
		at, bt := strings.ToLower(firstTag(a)), strings.ToLower(firstTag(b))
		if c := strings.Compare(at, bt); c != 0 {
			return c
		}
	default:
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
	}
	return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
}

func apiLess(a, b API, key string, asc bool) bool {
	at, bt := strings.ToLower(firstTag(a)), strings.ToLower(firstTag(b))
	if key == "tag" && (at == "") != (bt == "") {
		return bt == ""
	}
	c := compareAPIs(a, b, key)
	if !asc {
		c = -c
	}
	return c < 0
}

func sortAPIs(items []API, key string, asc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		return apiLess(items[i], items[j], key, asc)
	})
}
