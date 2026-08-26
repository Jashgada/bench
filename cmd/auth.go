package cmd

import (
	"fmt"
	"net/http"
	"strings"
)

type securityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme"`
	Name   string `json:"name"`
	In     string `json:"in"`
}

// lookupCredential returns the first non-empty value among candidate keys.
// Credentials are looked up by scheme name first, then well-known aliases.
func lookupCredential(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := values[key]; value != "" {
			return value
		}
	}
	return ""
}

// applySecurity injects credentials for the first satisfied security
// requirement of the operation. Explicit user-supplied headers always win.
// It returns warnings describing credentials that could not be resolved.
func applySecurity(request *http.Request, project Project, api API, values map[string]string) []string {
	var warnings []string
	for _, schemeName := range api.Security {
		scheme, ok := project.SecuritySchemes[schemeName]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("unknown security scheme %q", schemeName))
			continue
		}
		switch scheme.Type {
		case "http":
			switch scheme.Scheme {
			case "bearer":
				token := lookupCredential(values, schemeName, "token", "bearer_token", "access_token")
				if token == "" {
					warnings = append(warnings, fmt.Sprintf("no credential for bearer scheme %q", schemeName))
					continue
				}
				if request.Header.Get("Authorization") == "" {
					request.Header.Set("Authorization", "Bearer "+token)
				}
				return warnings
			case "basic":
				username := lookupCredential(values, schemeName+"_username", "username")
				password := lookupCredential(values, schemeName+"_password", "password")
				if username == "" || password == "" {
					warnings = append(warnings, fmt.Sprintf("no credential for basic scheme %q", schemeName))
					continue
				}
				if request.Header.Get("Authorization") == "" {
					request.SetBasicAuth(username, password)
				}
				return warnings
			default:
				warnings = append(warnings, fmt.Sprintf("unsupported http scheme %q for %q", scheme.Scheme, schemeName))
			}
		case "apiKey":
			headerName := scheme.Name
			if headerName == "" || !strings.EqualFold(scheme.In, "header") {
				warnings = append(warnings, fmt.Sprintf("api key scheme %q is not header-based", schemeName))
				continue
			}
			value := lookupCredential(values, schemeName, strings.ToLower(headerName), "api_key")
			if value == "" {
				warnings = append(warnings, fmt.Sprintf("no credential for api key scheme %q", schemeName))
				continue
			}
			if request.Header.Get(headerName) == "" {
				request.Header.Set(headerName, value)
			}
			return warnings
		default:
			warnings = append(warnings, fmt.Sprintf("unsupported security type %q for %q", scheme.Type, schemeName))
		}
	}
	return warnings
}
