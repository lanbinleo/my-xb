package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"myxb/internal/api"
	"os"
	"path/filepath"
	"strings"
)

func getCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(yellow("Username: "))
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print(yellow("Password: "))
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)

	fmt.Println()
	return user, pass
}

// loginFunc is a function type that performs the actual login API call
type loginFunc func(username, captchaCode string) error

func performLogin(apiClient *api.API, username, password string) error {
	loginFn := func(user, captcha string) error {
		return apiClient.Login(user, password, captcha)
	}
	return performLoginWithCaptcha(apiClient, username, loginFn, true)
}

func performLoginWithHash(apiClient *api.API, username, passwordHash string) error {
	loginFn := func(user, captcha string) error {
		return apiClient.LoginWithPasswordHash(user, passwordHash, captcha)
	}
	return performLoginWithCaptcha(apiClient, username, loginFn, false)
}

// performLoginWithCaptcha handles the common login flow with captcha handling
func performLoginWithCaptcha(apiClient *api.API, username string, loginFn loginFunc, allowInteractiveCaptcha bool) error {
	captchaResp, err := apiClient.GetCaptcha()
	if err != nil {
		return fmt.Errorf("failed to get captcha: %w", err)
	}

	var captchaCode string

	if captchaResp.Data != "" {
		if !allowInteractiveCaptcha {
			// For auto-login with saved credentials, we can't handle captcha interactively
			return fmt.Errorf("captcha required, please run 'myxb login' again")
		}

		// Save captcha image to temp file
		captchaPath, err := saveCaptcha(captchaResp.Data)
		if err != nil {
			printWarning(fmt.Sprintf("Failed to save captcha image: %v", err))
			printInfo("Captcha data received but could not be saved.")
		} else {
			printInfo(fmt.Sprintf("Captcha saved to: %s", captchaPath))
		}

		// Prompt for captcha code
		fmt.Print(cyan("Enter captcha code: "))
		reader := bufio.NewReader(os.Stdin)
		captchaCode, _ = reader.ReadString('\n')
		captchaCode = strings.TrimSpace(captchaCode)
	} else {
		captchaCode = ""
	}

	printInfo("Logging in...")
	return loginFn(username, captchaCode)
}

func saveCaptcha(base64Data string) (string, error) {
	// Remove the data URL prefix if present
	base64Data = strings.TrimPrefix(base64Data, "data:image/png;base64,")

	// Decode base64
	imgData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	// Save to temp directory
	tempDir := os.TempDir()
	captchaPath := filepath.Join(tempDir, "myxb_captcha.png")

	err = os.WriteFile(captchaPath, imgData, 0644)
	if err != nil {
		return "", err
	}

	return captchaPath, nil
}
