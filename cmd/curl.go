package cmd

import (
	"net/url"
	"strings"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func requestURL(project Project, api API, params map[string]string) string {
	path := api.Path
	for _, p := range api.PathParams {
		if val := params[p.Name]; val != "" {
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(val))
		}
	}
	raw := strings.TrimRight(project.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	for _, p := range api.QueryParams {
		if val := params[p.Name]; val != "" {
			query.Set(p.Name, val)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func curlCommand(project Project, api API, params map[string]string, body []byte) string {
	var b strings.Builder
	b.WriteString("curl")
	method := strings.ToUpper(api.Method)
	if method != "" && method != "GET" {
		b.WriteString(" -X " + method)
	}
	b.WriteString(" " + shellQuote(requestURL(project, api, params)))
	hasContentType := false
	for _, h := range api.Headers {
		val := params[h.Name]
		if val == "" {
			continue
		}
		b.WriteString(" -H " + shellQuote(h.Name+": "+val))
		if strings.EqualFold(h.Name, "Content-Type") {
			hasContentType = true
		}
	}
	if len(body) > 0 {
		if !hasContentType {
			b.WriteString(" -H " + shellQuote("Content-Type: application/json"))
		}
		b.WriteString(" -d " + shellQuote(string(body)))
	}
	return b.String()
}

func (m *tuiModel) copyCurl() {
	api := m.currentAPI()
	params, body, err := formRequest(m.fields)
	if err != nil {
		m.copyStatus = "Curl failed: " + err.Error()
		return
	}
	if err := copyToClipboard([]byte(curlCommand(m.project, api, params, body))); err != nil {
		m.copyStatus = "Copy failed: " + err.Error()
		return
	}
	m.copyStatus = "Copied curl command"
}
