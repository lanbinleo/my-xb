package main

import (
	"fmt"
	"myxb/internal/api"
	"myxb/internal/client"
	"myxb/internal/config"
	"os"
)

var version = "1.0.0"

func main() {
	// Parse command and flags
	args := os.Args[1:]
	command := ""
	showTasks := false

	// Parse flags
	filteredArgs := []string{}
	for _, arg := range args {
		if arg == "-t" || arg == "--tasks" {
			showTasks = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) > 0 {
		command = filteredArgs[0]
	}

	switch command {
	case "login":
		runLogin()
	case "logout":
		runLogout()
	case "help", "-h", "--help":
		printHelp()
	default:
		// Default: run GPA calculation
		runGPA(showTasks)
	}
}

func printHelp() {
	printBanner(version)

	fmt.Println("Usage:")
	fmt.Println("  myxb           Calculate GPA (requires login)")
	fmt.Println("  myxb -t        Show detailed task breakdown for each subject")
	fmt.Println("  myxb login     Login to save credentials")
	fmt.Println("  myxb logout    Clear saved credentials")
	fmt.Println("  myxb help      Show this help message")
	fmt.Println()
}

func ensureLogin() (*api.API, error) {
	// Check if already logged in
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg != nil {
		printInfo(fmt.Sprintf("Using saved credentials for: %s", cyan(cfg.Username)))

		// Create HTTP client
		httpClient, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}

		apiClient := api.New(httpClient)

		// Try to login with saved credentials
		// We need to hash the already-hashed password one more time
		if err := performLoginWithHash(apiClient, cfg.Username, cfg.PasswordHash); err != nil {
			printWarning("Saved credentials failed, please login again")
			config.Delete()
			return nil, fmt.Errorf("authentication failed")
		}

		return apiClient, nil
	}

	return nil, fmt.Errorf("not logged in")
}

func runGPA(showTasks bool) {
	printBanner(version)

	apiClient, err := ensureLogin()
	if err != nil {
		printError("You need to login first")
		fmt.Println()
		printInfo("Run: myxb login")
		os.Exit(1)
	}

	printSuccess("Authentication successful!")
	fmt.Println()

	calculateGPA(apiClient, showTasks)
}

func runLogin() {
	printBanner(version)

	fmt.Println("Your credentials will be saved " + bold(cyan("locally")) + " for future use.")
	fmt.Println()

	// Create HTTP client
	httpClient, err := client.New()
	if err != nil {
		printError(fmt.Sprintf("Failed to create HTTP client: %v", err))
		os.Exit(1)
	}

	apiClient := api.New(httpClient)

	// Get credentials
	username, password := getCredentials()

	// Login
	if err := performLogin(apiClient, username, password); err != nil {
		printError(fmt.Sprintf("Login failed: %v", err))
		os.Exit(1)
	}

	printSuccess("Login successful!")

	// Save credentials
	cfg := &config.Config{
		Username:     username,
		PasswordHash: config.HashPassword(password),
	}

	if err := config.Save(cfg); err != nil {
		printWarning(fmt.Sprintf("Failed to save credentials: %v", err))
	} else {
		printSuccess("Credentials saved!")
		configPath, _ := config.GetConfigPath()
		fmt.Println(gray(fmt.Sprintf(" - Config saved to: %s", configPath)))
	}

	fmt.Println()
	printInfo("You can now run 'myxb' to calculate your GPA")
}

func runLogout() {
	printBanner(version)

	// Check if there are saved credentials
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		printWarning("No saved credentials found")
		os.Exit(0)
	}

	// Delete the config file
	if err := config.Delete(); err != nil {
		printError(fmt.Sprintf("Failed to delete credentials: %v", err))
		os.Exit(1)
	}

	printSuccess("Credentials cleared!")
	configPath, _ := config.GetConfigPath()
	fmt.Println(gray(fmt.Sprintf(" - Config file removed: %s", configPath)))
	fmt.Println()
	printInfo("Run 'myxb login' to login again")
}
