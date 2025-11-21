package main

import (
	"context"
	"fmt"
	"myxb/internal/api"
	"myxb/internal/client"
	"myxb/internal/config"
	"os"

	"github.com/urfave/cli/v3"
)

var version = "Dev"

func main() {
	cmd := &cli.Command{
		Name:    "myxb",
		Usage:   "GPA Calculator & Score Tracker for Xiaobao",
		Version: version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "tasks",
				Aliases: []string{"t"},
				Usage:   "Show detailed task information for each subject",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			showTasks := c.Bool("tasks")
			runGPA(showTasks)
			return nil
		},
		Commands: []*cli.Command{
			{

				Name:  "login",
				Usage: "Login and save credentials",
				Action: func(ctx context.Context, c *cli.Command) error {
					runLogin()
					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "Clear saved credentials",
				Action: func(ctx context.Context, c *cli.Command) error {
					runLogout()
					return nil
				},
			},
			{
				Name:  "update",
				Usage: "Check for updates and update to the latest version",
				Action: func(ctx context.Context, c *cli.Command) error {
					runUpdate()
					return nil
				},
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			printBanner(version)
			return ctx, nil
		},
		After: func(ctx context.Context, c *cli.Command) error {
			// 自动检查更新（仅在非 update 命令时）
			if c.Name != "update" {
				checkForUpdates()
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		printError(err.Error())
		os.Exit(1)
	}
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
