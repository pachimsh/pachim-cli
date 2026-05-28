package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	mirrorBase     = "https://mirrors.pachim.app/cli"
	githubRepo     = "pachimsh/pachim-cli"
	checkCacheFile = "update-check.json"
	cacheTTL       = 1 * time.Hour
	httpTimeout    = 15 * time.Second
)

type checkCache struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checked_at"`
}

// MaybePromptAndUpdate checks for a newer release and optionally self-updates.
// On successful update it re-executes the same command and does not return.
func MaybePromptAndUpdate(currentVersion string, args []string) error {
	if currentVersion == "" || currentVersion == "dev" {
		return nil
	}

	if os.Getenv("PACHIM_SKIP_UPDATE") == "1" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		return nil
	}

	if !isNewer(currentVersion, latest) {
		return nil
	}

	fmt.Println()
	color.Yellow("⚠ A new version of pachim is available: %s (you have %s)", latest, currentVersion)
	if runtime.GOOS == "windows" {
		if exe, err := os.Executable(); err == nil && isMSIInstallPath(exe) {
			fmt.Println("  This update may request administrator permission (MSI install).")
		}
	}
	fmt.Print("Would you like to update now? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println()
		return nil
	}

	fmt.Println()
	color.Cyan("⟳ Updating pachim to %s...", latest)

	if err := applyUpdate(ctx, latest, args); err != nil {
		color.Red("Update failed: %s", err)
		fmt.Println("Continuing with the current version.")
		fmt.Println()
		return nil
	}

	color.Green("✓ Updated to %s", latest)
	fmt.Println("⟳ Restarting command...")
	fmt.Println()

	restart(args)
	return nil
}

func fetchLatestVersion(ctx context.Context) (string, error) {
	if cached, ok := readCache(); ok {
		return cached, nil
	}

	latest, err := fetchLatestFromRemote(ctx)
	if err != nil {
		return "", err
	}

	writeCache(latest)

	return latest, nil
}

func fetchLatestFromRemote(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mirrorBase+"/latest.txt", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr == nil {
			if v := strings.TrimSpace(string(body)); v != "" {
				return normalizeVersion(v), nil
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	githubURL := "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, githubURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.TagName == "" {
		return "", fmt.Errorf("empty tag_name from GitHub")
	}

	return normalizeVersion(payload.TagName), nil
}

func isNewer(current, latest string) bool {
	current = normalizeVersion(current)
	latest = normalizeVersion(latest)

	return compareSemver(current, latest) < 0
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func compareSemver(a, b string) int {
	pa, oka := parseSemver(a)
	pb, okb := parseSemver(b)

	if !oka || !okb {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}

	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}

	return 0
}

func parseSemver(v string) ([3]int, bool) {
	var out [3]int

	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" {
		return out, false
	}

	// Remove pre-release/build metadata for comparison.
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return out, false
	}

	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			out[i] = 0
			continue
		}

		n := 0
		if _, err := fmt.Sscanf(parts[i], "%d", &n); err != nil {
			return out, false
		}
		out[i] = n
	}

	return out, true
}

func readCache() (string, bool) {
	path, err := cachePath()
	if err != nil {
		return "", false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	var cache checkCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", false
	}

	if cache.Latest == "" || time.Since(cache.CheckedAt) > cacheTTL {
		return "", false
	}

	return cache.Latest, true
}

func writeCache(latest string) {
	path, err := cachePath()
	if err != nil {
		return
	}

	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0700)

	cache := checkCache{
		Latest:    latest,
		CheckedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0600)
}

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pachim", checkCacheFile), nil
}

func applyUpdate(ctx context.Context, version string, args []string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	if goarch == "386" {
		return fmt.Errorf("unsupported architecture: 386")
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}

	if goos == "windows" && isMSIInstallPath(executable) {
		return applyMSIUpdate(ctx, version, goarch, executable, args)
	}

	tmpDir, err := os.MkdirTemp("", "pachim-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	binaryName := "pachim"
	if goos == "windows" {
		binaryName = "pachim.exe"
	}

	archivePath := filepath.Join(tmpDir, "release-archive")
	newBinaryPath := filepath.Join(tmpDir, binaryName)

	if err := downloadRelease(ctx, version, goos, goarch, archivePath); err != nil {
		return err
	}

	if err := extractBinary(archivePath, goos, newBinaryPath); err != nil {
		return err
	}

	if err := os.Chmod(newBinaryPath, 0755); err != nil && goos != "windows" {
		return err
	}

	if goos == "windows" {
		return applyUpdateWindows(executable, newBinaryPath, args)
	}

	return replaceExecutable(executable, newBinaryPath)
}

func applyMSIUpdate(ctx context.Context, version, goarch, executable string, args []string) error {
	tmpDir, err := os.MkdirTemp("", "pachim-msi-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	msiName := msiFilename(goarch)
	msiPath := filepath.Join(tmpDir, msiName)
	msiURLs := []string{
		fmt.Sprintf("%s/%s/%s", mirrorBase, version, msiName),
		fmt.Sprintf("%s/%s", mirrorBase, msiName),
		fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", githubRepo, version, msiName),
	}

	if err := downloadFromURLs(ctx, msiURLs, msiPath); err != nil {
		return fmt.Errorf("failed to download MSI: %w", err)
	}

	if err := runMSIInstaller(msiPath); err != nil {
		return err
	}

	// Restart command using the same executable path after MSI upgrade.
	restart(args)
	return nil
}

func msiFilename(goarch string) string {
	if goarch == "arm64" {
		return "pachim_windows_arm64.msi"
	}

	return "pachim_windows_amd64.msi"
}

func downloadRelease(ctx context.Context, version, goos, goarch, dest string) error {
	filename := archiveFilename(version, goos, goarch)
	urls := []string{
		fmt.Sprintf("%s/%s/%s", mirrorBase, version, filename),
		fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", githubRepo, version, filename),
	}

	return downloadFromURLs(ctx, urls, dest)
}

func downloadFromURLs(ctx context.Context, urls []string, dest string) error {
	client := &http.Client{Timeout: httpTimeout}

	var lastErr error
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("%s returned %d", url, resp.StatusCode)
			continue
		}

		f, err := os.Create(dest)
		if err != nil {
			resp.Body.Close()
			return err
		}

		_, copyErr := io.Copy(f, resp.Body)
		closeErr := f.Close()
		resp.Body.Close()

		if copyErr != nil {
			lastErr = copyErr
			continue
		}
		if closeErr != nil {
			lastErr = closeErr
			continue
		}

		return nil
	}

	return fmt.Errorf("download failed: %w", lastErr)
}

func isMSIInstallPath(executable string) bool {
	clean := strings.ToLower(filepath.Clean(executable))

	return strings.Contains(clean, `\program files\pachim\`) ||
		strings.Contains(clean, `\program files (x86)\pachim\`)
}

func runMSIInstaller(msiPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("MSI installer is only supported on Windows")
	}

	escapedPath := strings.ReplaceAll(msiPath, `'`, `''`)
	psScript := fmt.Sprintf(
		"$msi='%s'; Start-Process msiexec.exe -Verb RunAs -Wait -ArgumentList @('/i', $msi, '/passive', '/norestart')",
		escapedPath,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("msi update failed: %w", err)
	}

	return nil
}

func archiveFilename(version, goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("pachim_%s_%s.zip", goos, goarch)
	}

	return fmt.Sprintf("pachim_%s_%s.tar.gz", goos, goarch)
}

func extractBinary(archivePath, goos, dest string) error {
	if goos == "windows" {
		return extractZip(archivePath, dest)
	}

	return extractTarGz(archivePath, dest)
}

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		name := filepath.Base(header.Name)
		if name != "pachim" {
			continue
		}

		out, err := os.Create(dest)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}

		return out.Close()
	}

	return fmt.Errorf("pachim binary not found in archive")
}

func extractZip(archivePath, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		if filepath.Base(file.Name) != "pachim.exe" {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		closeOut := out.Close()
		rc.Close()

		if copyErr != nil {
			return copyErr
		}

		return closeOut
	}

	return fmt.Errorf("pachim.exe not found in archive")
}

func replaceExecutable(target, source string) error {
	if err := copyFile(source, target+".new"); err != nil {
		return err
	}

	if err := os.Rename(target+".new", target); err != nil {
		_ = os.Remove(target + ".new")
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}

func applyUpdateWindows(executable, newBinary string, originalArgs []string) error {
	batchPath := filepath.Join(os.TempDir(), fmt.Sprintf("pachim-update-%d.bat", time.Now().UnixNano()))

	argsLine := ""
	if len(originalArgs) > 1 {
		parts := make([]string, 0, len(originalArgs)-1)
		for _, arg := range originalArgs[1:] {
			parts = append(parts, quoteWindowsArg(arg))
		}
		argsLine = strings.Join(parts, " ")
	}

	script := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
move /y "%s" "%s" >nul
start "" "%s" %s
del "%%~f0"
`, newBinary, executable, executable, argsLine)

	if err := os.WriteFile(batchPath, []byte(script), 0600); err != nil {
		return err
	}

	cmd := exec.Command("cmd", "/C", "start", "", batchPath)
	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}

func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\"") {
		return arg
	}
	return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
}

func restart(args []string) {
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Restart failed: %v\n", err)
		os.Exit(0)
	}

	restartArgs := args
	if len(restartArgs) == 0 {
		restartArgs = []string{executable}
	} else {
		restartArgs = make([]string, len(args))
		copy(restartArgs, args)
		restartArgs[0] = executable
	}

	if err := syscallExec(executable, restartArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Restart failed: %v\n", err)
	}

	os.Exit(0)
}
