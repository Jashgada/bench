package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func (m *tuiModel) copyResponse() {
	r := m.displayedResult()
	if r == nil {
		return
	}
	data := r.Body
	if len(data) > 0 {
		var pretty strings.Builder
		if err := jsonIndent(&pretty, data); err == nil {
			data = []byte(pretty.String())
		}
	}
	if err := copyToClipboard(data); err != nil {
		m.copyStatus = "Copy failed: " + err.Error()
		return
	}
	m.copyStatus = "Copied response to clipboard"
}

func copyToClipboard(data []byte) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(string(data))
	return cmd.Run()
}
