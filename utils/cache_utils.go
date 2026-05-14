package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".mangahub")
}
func SaveUDPServerAddr(addr string) error {
	return os.WriteFile(
		filepath.Join(configDir(), "udp_server"),
		[]byte(addr),
		0600,
	)
}
func LoadServerIPAddr() (string, error) {
	data, err := os.ReadFile(
		filepath.Join(configDir(), "udp_server"),
	)
	if err != nil {
		return "", err
	}
	addr := strings.Split(string(data), ":")[0]
	return strings.TrimSpace(addr), nil
}

func LoadUDPServerAddr() (string, error) {
	data, err := os.ReadFile(
		filepath.Join(configDir(), "udp_server"),
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
