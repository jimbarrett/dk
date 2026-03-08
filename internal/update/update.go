package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ReleaseInfo holds the result of checking GitHub for updates.
type ReleaseInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	DownloadURL     string
	PublishedAt     time.Time
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check queries GitHub for the latest release and compares it to currentVersion.
func Check(currentVersion string) (*ReleaseInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/jimbarrett/dk/releases/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "dk-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	info := &ReleaseInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
		PublishedAt:    release.PublishedAt,
	}

	assetName := AssetName(release.TagName)
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	info.UpdateAvailable = CompareVersions(currentVersion, release.TagName)

	return info, nil
}

// Apply downloads the binary from downloadURL and replaces the running binary.
func Apply(downloadURL string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "dk-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "dk-updater")

	resp, err := client.Do(req)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing binary: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// AssetName returns the expected release asset filename for the current platform.
func AssetName(version string) string {
	v := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("dk_%s_%s_%s", v, runtime.GOOS, runtime.GOARCH)
}

// CompareVersions returns true if latest is newer than current.
// If current is "dev", always returns true.
func CompareVersions(current, latest string) bool {
	if current == "dev" {
		return true
	}

	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}

// CanWriteBinary returns the resolved path of the running binary and whether it can be replaced.
func CanWriteBinary() (string, bool) {
	execPath, err := os.Executable()
	if err != nil {
		return "", false
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return execPath, false
	}

	dir := filepath.Dir(execPath)
	f, err := os.CreateTemp(dir, ".dk-write-test-*")
	if err != nil {
		return execPath, false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return execPath, true
}

// ManualUpdateCommand returns a shell command string for manually updating the binary.
func ManualUpdateCommand(downloadURL, binaryPath string) string {
	return fmt.Sprintf("curl -L %s -o /tmp/dk && chmod +x /tmp/dk && sudo mv /tmp/dk %s", downloadURL, binaryPath)
}
