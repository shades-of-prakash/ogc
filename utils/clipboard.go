package utils

import (
	"fmt"
	"os"
	"os/exec"
)
func GetClipboard() (string, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		out, err := exec.Command("wl-paste", "-n").Output()
		return string(out), err
	}

	if os.Getenv("DISPLAY") != "" {
		out, err := exec.Command(
			"xclip",
			"-selection",
			"clipboard",
			"-o",
		).Output()
		return string(out), err
	}

	return "", fmt.Errorf("could not detect X11 or Wayland")
}

func SetClipboard(content string) error {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		cmd := exec.Command("wl-copy")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			defer stdin.Close()
			_, _ = stdin.Write([]byte(content))
		}()
		return cmd.Run()
	}

	if os.Getenv("DISPLAY") != "" {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			defer stdin.Close()
			_, _ = stdin.Write([]byte(content))
		}()
		return cmd.Run()
	}

	return fmt.Errorf("could not detect X11 or Wayland")
}
