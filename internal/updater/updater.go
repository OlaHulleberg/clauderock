package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubAPIURL  = "https://api.github.com/repos/OlaHulleberg/clauderock/releases/latest"
	githubRepoURL = "https://github.com/OlaHulleberg/clauderock"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckForUpdates checks for updates in the background and notifies the user
func CheckForUpdates(currentVersion string) {
	if currentVersion == "dev" {
		return // Skip update check for development builds
	}

	latestVersion, err := getLatestVersion()
	if err != nil {
		// Silently fail - don't interrupt the user's workflow
		return
	}

	if latestVersion != currentVersion && latestVersion != "" {
		fmt.Fprintf(os.Stderr, "\n⚠️  New version available: %s (current: %s)\n", latestVersion, currentVersion)
		fmt.Fprintf(os.Stderr, "   Run 'clauderock manage update' to upgrade\n\n")
	}
}

// Update checks for and installs the latest version
func Update(currentVersion string) error {
	if currentVersion == "dev" {
		return fmt.Errorf("cannot update development build")
	}

	fmt.Println("Checking for updates...")

	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := release.TagName
	if latestVersion == currentVersion {
		fmt.Printf("Already on latest version: %s\n", currentVersion)
		return nil
	}

	fmt.Printf("New version available: %s (current: %s)\n", latestVersion, currentVersion)

	// Find the appropriate binary for the current platform
	assetName := getBinaryAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("Downloading %s...\n", assetName)
	if err := downloadAndReplace(downloadURL); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to version %s\n", latestVersion)
	return nil
}

func getLatestVersion() (string, error) {
	release, err := getLatestRelease()
	if err != nil {
		return "", err
	}
	return release.TagName, nil
}

func getLatestRelease() (*GitHubRelease, error) {
	resp, err := http.Get(githubAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func getBinaryAssetName() string {
	// Expected archive format from GoReleaser
	// tar.gz for linux/darwin, zip for windows
	// Examples: clauderock_darwin_arm64.tar.gz, clauderock_windows_amd64.zip
	os := runtime.GOOS
	arch := runtime.GOARCH

	name := fmt.Sprintf("clauderock_%s_%s", os, arch)
	if os == "windows" {
		name += ".zip"
	} else {
		name += ".tar.gz"
	}
	return name
}

func downloadAndReplace(url string) error {
	// Download the archive
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create a temporary file for the archive
	tmpFile, err := os.CreateTemp("", "clauderock-archive-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write the downloaded archive to the temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Extract the binary from the archive
	var binaryPath string
	if strings.HasSuffix(url, ".zip") {
		binaryPath, err = extractFromZip(tmpPath)
	} else if strings.HasSuffix(url, ".tar.gz") {
		binaryPath, err = extractFromTarGz(tmpPath)
	} else {
		return fmt.Errorf("unsupported archive format")
	}
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(binaryPath)

	// Make it executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return err
	}

	// Get the current executable path
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Replace the current binary with the new one
	if runtime.GOOS == "windows" {
		backupPath := currentPath + ".old"
		if err := os.Rename(currentPath, backupPath); err != nil {
			return err
		}
		if err := os.Rename(binaryPath, currentPath); err != nil {
			os.Rename(backupPath, currentPath)
			return err
		}
		os.Remove(backupPath)
	} else {
		if err := os.Rename(binaryPath, currentPath); err != nil {
			return err
		}
	}

	return nil
}

func extractFromTarGz(archivePath string) (string, error) {
	// Open the archive
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Find the clauderock binary in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Look for the binary (usually just "clauderock")
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "clauderock" {
			// Create temp file for the extracted binary
			tmpFile, err := os.CreateTemp("", "clauderock-binary-*")
			if err != nil {
				return "", err
			}
			tmpPath := tmpFile.Name()

			// Extract the binary
			if _, err := io.Copy(tmpFile, tarReader); err != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				return "", err
			}
			tmpFile.Close()

			return tmpPath, nil
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}

func extractFromZip(archivePath string) (string, error) {
	// Open the zip archive
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	// Find the clauderock.exe binary
	for _, file := range zipReader.File {
		if filepath.Base(file.Name) == "clauderock.exe" {
			// Open the file in the archive
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			// Create temp file for the extracted binary
			tmpFile, err := os.CreateTemp("", "clauderock-binary-*.exe")
			if err != nil {
				return "", err
			}
			tmpPath := tmpFile.Name()

			// Extract the binary
			if _, err := io.Copy(tmpFile, rc); err != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				return "", err
			}
			tmpFile.Close()

			return tmpPath, nil
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}
