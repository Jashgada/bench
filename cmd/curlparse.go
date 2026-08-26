package cmd

import (
	"fmt"
	"strings"
)

// tokenizeShell splits a shell-like command line, respecting single quotes,
// double quotes, and backslash escapes.
func tokenizeShell(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	hasToken := false
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch c {
		case ' ', '\t':
			if hasToken {
				tokens = append(tokens, current.String())
				current.Reset()
				hasToken = false
			}
		case '\'':
			hasToken = true
			end := strings.IndexByte(input[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("unterminated single quote")
			}
			current.WriteString(input[i+1 : i+1+end])
			i += end + 1
		case '"':
			hasToken = true
			i++
			closed := false
			for ; i < len(input); i++ {
				if input[i] == '\\' && i+1 < len(input) {
					switch input[i+1] {
					case '"', '\\', '$', '`':
						current.WriteByte(input[i+1])
					default:
						current.WriteByte('\\')
						current.WriteByte(input[i+1])
					}
					i++
					continue
				}
				if input[i] == '"' {
					closed = true
					break
				}
				current.WriteByte(input[i])
			}
			if !closed {
				return nil, fmt.Errorf("unterminated double quote")
			}
		case '\\':
			hasToken = true
			if i+1 < len(input) {
				i++
				current.WriteByte(input[i])
			}
		default:
			hasToken = true
			current.WriteByte(c)
		}
	}
	if hasToken {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}

type curlRequest struct {
	Method  string
	URL     string
	Headers [][2]string
	Body    string
}

// parseCurlTokens understands the subset of curl flags bench needs:
// -X/--request, -H/--header, -d/--data/--data-raw/--data-binary, --url,
// and a bare URL. Unknown flags are skipped; -d implies POST.
func parseCurlTokens(tokens []string) (curlRequest, error) {
	request := curlRequest{Method: "GET"}
	urlSeen := false
	dataSeen := false
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-X", "--request":
			if i+1 >= len(tokens) {
				return request, fmt.Errorf("%s requires a method", token)
			}
			i++
			request.Method = strings.ToUpper(tokens[i])
		case "-H", "--header":
			if i+1 >= len(tokens) {
				return request, fmt.Errorf("%s requires a header", token)
			}
			i++
			name, value, found := strings.Cut(tokens[i], ":")
			if !found {
				return request, fmt.Errorf("invalid header %q", tokens[i])
			}
			request.Headers = append(request.Headers, [2]string{strings.TrimSpace(name), strings.TrimSpace(value)})
		case "-d", "--data", "--data-raw", "--data-binary":
			if i+1 >= len(tokens) {
				return request, fmt.Errorf("%s requires data", token)
			}
			i++
			request.Body = tokens[i]
			dataSeen = true
		case "--url":
			if i+1 >= len(tokens) {
				return request, fmt.Errorf("%s requires a URL", token)
			}
			i++
			request.URL = tokens[i]
			urlSeen = true
		default:
			if strings.HasPrefix(token, "-") {
				continue
			}
			if !urlSeen && (strings.HasPrefix(token, "http://") || strings.HasPrefix(token, "https://")) {
				request.URL = token
				urlSeen = true
			}
		}
	}
	if request.URL == "" {
		return request, fmt.Errorf("no URL found")
	}
	if dataSeen && request.Method == "GET" {
		request.Method = "POST"
	}
	return request, nil
}

func parseCurl(input string) (curlRequest, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "curl")
	tokens, err := tokenizeShell(input)
	if err != nil {
		return curlRequest{}, err
	}
	return parseCurlTokens(tokens)
}
