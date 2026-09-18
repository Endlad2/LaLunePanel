package protocol

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// EnsureBinary downloads the protocol server binary into <dataDir>/bin if it
// isn't already present, and returns its absolute path.
func EnsureBinary(dataDir string, p Protocol) (string, error) {
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}

	dest := filepath.Join(binDir, p.Binary.Binary)
	if fi, err := os.Stat(dest); err == nil && fi.Size() > 0 {
		return dest, nil
	}

	url, err := p.URLForArch()
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	f.Close()

	if err := os.Chmod(tmp, 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}
