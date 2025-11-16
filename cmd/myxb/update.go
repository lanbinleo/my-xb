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

// runUpdate 执行更新操作
func runUpdate(silentCheck bool) {
	if !silentCheck {
		printInfo("正在检查更新...")
		fmt.Println()
	}

	latest, found, err := selfupdate.DetectLatest(fmt.Sprintf("%s/%s", githubOwner, githubRepo))
	if err != nil {
		if !silentCheck {
			printError(fmt.Sprintf("检查更新失败: %v", err))
		}
		return
	}

	currentVersion := version
	// 移除可能的 v 前缀
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	current, err := semver.Parse(currentVersion)
	if err != nil {
		if !silentCheck {
			printError(fmt.Sprintf("解析当前版本号失败: %v", err))
		}
		return
	}

	if !found || latest.Version.LTE(current) {
		if !silentCheck {
			printSuccess("已是最新版本!")
			fmt.Println(gray(fmt.Sprintf("  当前版本: %s", currentVersion)))
		}
		return
	}

	if silentCheck {
		// 静默检查只显示提示信息
		return
	}

	// 显示更新信息
	fmt.Println(green("✓") + " 发现新版本!")
	fmt.Println(gray(fmt.Sprintf("  当前版本: %s", currentVersion)))
	fmt.Println(gray(fmt.Sprintf("  最新版本: %s", latest.Version)))
	fmt.Println()

	// 执行更新
	printInfo("正在下载更新...")

	exe, err := os.Executable()
	if err != nil {
		printError(fmt.Sprintf("获取程序路径失败: %v", err))
		return
	}

	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		// 如果是 Windows，可能需要特殊处理
		if runtime.GOOS == "windows" {
			printWarning("无法直接替换文件，尝试使用批处理脚本更新...")
			if err := updateOnWindows(latest.AssetURL, exe); err != nil {
				printError(fmt.Sprintf("更新失败: %v", err))
				fmt.Println()
				printInfo("请手动下载最新版本:")
				fmt.Println(gray(fmt.Sprintf("  %s", latest.AssetURL)))
			}
		} else {
			printError(fmt.Sprintf("更新失败: %v", err))
			fmt.Println()
			printInfo("请手动下载最新版本:")
			fmt.Println(gray(fmt.Sprintf("  %s", latest.AssetURL)))
		}
		return
	}

	printSuccess("更新完成!")
	fmt.Println(gray(fmt.Sprintf("  版本: %s → %s", currentVersion, latest.Version)))
	fmt.Println()
	printInfo("请重新运行程序以使用新版本")
}

// updateOnWindows 在 Windows 上使用批处理脚本进行更新
func updateOnWindows(assetURL, exePath string) error {
	// 下载新版本到临时文件
	newExePath := exePath + ".new.exe"
	if err := selfupdate.UpdateTo(assetURL, newExePath); err != nil {
		return fmt.Errorf("下载新版本失败: %w", err)
	}

	// 创建批处理脚本
	batContent := fmt.Sprintf(`@echo off
echo 正在更新 MyXB...
timeout /t 2 /nobreak >nul
move /y "%s" "%s"
if errorlevel 1 (
    echo.
    echo 更新失败！
    echo 请手动替换文件:
    echo   新版本: %s
    echo   目标位置: %s
    pause
) else (
    echo.
    echo 更新完成！
    echo 您可以重新运行程序了。
    timeout /t 3
)
del "%%~f0"
`, newExePath, exePath, newExePath, exePath)

	// 保存批处理脚本到临时目录
	batPath := filepath.Join(os.TempDir(), "myxb_update.bat")
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		os.Remove(newExePath)
		return fmt.Errorf("创建批处理脚本失败: %w", err)
	}

	// 启动批处理脚本
	cmd := exec.Command("cmd", "/C", "start", "/min", batPath)
	if err := cmd.Start(); err != nil {
		os.Remove(newExePath)
		os.Remove(batPath)
		return fmt.Errorf("启动批处理脚本失败: %w", err)
	}

	printSuccess("批处理脚本已启动")
	fmt.Println(gray("  程序将在几秒后自动更新..."))
	fmt.Println()

	// 立即退出当前程序
	os.Exit(0)
	return nil
}

// checkForUpdates 检查是否有可用更新（仅提示，不执行更新）
func checkForUpdates() {
	// 静默检查更新
	latest, found, err := selfupdate.DetectLatest(fmt.Sprintf("%s/%s", githubOwner, githubRepo))
	if err != nil {
		// 静默失败，不显示错误
		return
	}

	currentVersion := version
	// 移除可能的 v 前缀
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	current, err := semver.Parse(currentVersion)
	if err != nil {
		// 静默失败，不显示错误
		return
	}

	if !found || latest.Version.LTE(current) {
		// 已是最新版本，不显示任何信息
		return
	}

	// 发现新版本，显示提示
	fmt.Println()
	printInfo(fmt.Sprintf("发现新版本 %s 可用！", cyan("v"+latest.Version.String())))
	fmt.Println(gray(fmt.Sprintf("  运行 '%s' 来更新", cyan("myxb update"))))
}
