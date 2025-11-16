package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const (
	githubOwner = "lanbinleo"
	githubRepo  = "my-xb"
)

// versionInfo contains version check results
type versionInfo struct {
	Latest  *selfupdate.Release
	Current semver.Version
	HasNew  bool
}

// getLatestVersion checks for the latest version and returns version information
func getLatestVersion() (*versionInfo, error) {
	latest, found, err := selfupdate.DetectLatest(fmt.Sprintf("%s/%s", githubOwner, githubRepo))
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	currentVersion := strings.TrimPrefix(version, "v")
	current, err := semver.Parse(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current version: %w", err)
	}

	hasNew := found && latest.Version.GT(current)

	return &versionInfo{
		Latest:  latest,
		Current: current,
		HasNew:  hasNew,
	}, nil
}

// runUpdate performs the update operation
func runUpdate() {
	printInfo("Checking for updates...")
	fmt.Println()

	verInfo, err := getLatestVersion()
	if err != nil {
		printError(err.Error())
		return
	}

	if !verInfo.HasNew {
		printSuccess("Already up to date!")
		fmt.Println(gray(fmt.Sprintf("  Current version: %s", verInfo.Current)))
		return
	}

	// Display update information
	fmt.Println(green("✓") + " New version available!")
	fmt.Println(gray(fmt.Sprintf("  Current version: %s", verInfo.Current)))
	fmt.Println(gray(fmt.Sprintf("  Latest version:  %s", verInfo.Latest.Version)))
	fmt.Println()

	// Perform update
	printInfo("Downloading update...")

	exe, err := os.Executable()
	if err != nil {
		printError(fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}

	if err := selfupdate.UpdateTo(verInfo.Latest.AssetURL, exe); err != nil {
		// Special handling for Windows
		if runtime.GOOS == "windows" {
			printWarning("Cannot replace file directly, trying batch script update...")
			if err := updateOnWindows(verInfo.Latest.AssetURL, exe); err != nil {
				printError(fmt.Sprintf("Update failed: %v", err))
				fmt.Println()
				printInfo("Please download the latest version manually:")
				fmt.Println(gray(fmt.Sprintf("  %s", verInfo.Latest.AssetURL)))
			}
		} else {
			printError(fmt.Sprintf("Update failed: %v", err))
			fmt.Println()
			printInfo("Please download the latest version manually:")
			fmt.Println(gray(fmt.Sprintf("  %s", verInfo.Latest.AssetURL)))
		}
		return
	}

	printSuccess("Update completed!")
	fmt.Println(gray(fmt.Sprintf("  Version: %s → %s", verInfo.Current, verInfo.Latest.Version)))
	fmt.Println()
	printInfo("Please restart the program to use the new version")
}

// updateOnWindows performs update on Windows using a batch script
func updateOnWindows(assetURL, exePath string) error {
	// Download new version to temporary file
	newExePath := exePath + ".new.exe"
	if err := selfupdate.UpdateTo(assetURL, newExePath); err != nil {
		return fmt.Errorf("failed to download new version: %w", err)
	}

	// Create batch script
	// Use ping for delay, waiting for main program to exit
	// Use start /b to asynchronously delete the script itself
	batContent := fmt.Sprintf(`@echo off
echo Updating MyXB...
ping 127.0.0.1 -n 3 >nul
move /y "%s" "%s"
if errorlevel 1 (
    echo.
    echo Update failed!
    echo Please replace the file manually:
    echo   New version: %s
    echo   Target location: %s
    pause
) else (
    echo.
    echo Update completed!
    echo You can now restart the program.
    timeout /t 3
)
start /b cmd /c del "%%~f0"
`, newExePath, exePath, newExePath, exePath)

	// Save batch script to temporary directory
	batPath := filepath.Join(os.TempDir(), "myxb_update.bat")
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		os.Remove(newExePath)
		return fmt.Errorf("failed to create batch script: %w", err)
	}

	// Start batch script
	cmd := exec.Command("cmd", "/C", "start", "/min", batPath)
	if err := cmd.Start(); err != nil {
		os.Remove(newExePath)
		os.Remove(batPath)
		return fmt.Errorf("failed to start batch script: %w", err)
	}

	printSuccess("Batch script started")
	fmt.Println(gray("  Program will be updated in a few seconds..."))
	fmt.Println()

	// Exit current program immediately
	os.Exit(0)
	return nil
}

// checkForUpdates checks if an update is available (notification only, does not perform update)
func checkForUpdates() {
	verInfo, err := getLatestVersion()
	if err != nil {
		// Silent failure, do not display error
		return
	}

	if !verInfo.HasNew {
		// Already up to date, do not display any message
		return
	}

	// New version found, display notification
	fmt.Println()
	printInfo(fmt.Sprintf("New version %s available!", cyan("v"+verInfo.Latest.Version.String())))
	fmt.Println(gray(fmt.Sprintf("  Run '%s' to update", cyan("myxb update"))))
}
