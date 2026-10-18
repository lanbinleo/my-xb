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

var version = "1.3.0"

// skipUpdateCheck is set by commands that should not trigger the automatic
// update notice (update itself, or machine-readable --clean output).
var skipUpdateCheck bool

func main() {
	cmd := &cli.Command{
		Name:    "myxb",
		Usage:   "GPA Calculator & Score Tracker for Xiaobao",
		Version: version,
		Description: "Running 'myxb' with no command calculates your GPA (same as 'myxb gpa').\n" +
			"See 'myxb gpa --help' for GPA options and 'myxb week --help' for timetable options.",
		// Root keeps the GPA flags so 'myxb -t' and friends keep working, but
		// hides them: they belong to 'myxb gpa --help'.
		Flags: gpaFlags(true),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runGPACommand(c)
		},
		Commands: append([]*cli.Command{
			{
				Name:    "gpa",
				Aliases: []string{"g"},
				Usage:   "Calculate GPA and show scores (default when no command is given)",
				Flags:   gpaFlags(false),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runGPACommand(c)
				},
			},
			{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "Login and save credentials",
				Action: func(ctx context.Context, c *cli.Command) error {
					printBanner(version)
					runLogin()
					return nil
				},
			},
			{
				Name:    "logout",
				Aliases: []string{"lo"},
				Usage:   "Clear saved credentials",
				Action: func(ctx context.Context, c *cli.Command) error {
					runLogout()
					return nil
				},
			},
			{
				Name:    "update",
				Aliases: []string{"u"},
				Usage:   "Check for updates and update to the latest version",
				Action: func(ctx context.Context, c *cli.Command) error {
					skipUpdateCheck = true
					printBanner(version)
					runUpdate()
					return nil
				},
			},
			newScheduleCommand(),
			newAttendanceCommand(),
		}, newScheduleShortcutCommands()...),
		After: func(ctx context.Context, c *cli.Command) error {
			if !skipUpdateCheck {
				checkForUpdates()
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), normalizeCLIArgs(os.Args)); err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

func gpaFlags(hidden bool) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "formatted",
			Aliases: []string{"f"},
			Usage:   "Output format only: table, plain, markdown, or json (bare -f defaults to table)",
			Hidden:  hidden,
			Local:   true,
		},
		&cli.BoolFlag{
			Name:    "tasks",
			Aliases: []string{"t"},
			Usage:   "Show detailed task information for each subject",
			Hidden:  hidden,
			Local:   true,
		},
		&cli.BoolFlag{
			Name:   "refresh-cache",
			Usage:  "Rebuild cached task detail metadata used by --tasks",
			Hidden: hidden,
			Local:  true,
		},
		&cli.BoolFlag{
			Name:    "clean",
			Aliases: []string{"c"},
			Usage:   "Suppress banner, prompts, progress, and other human-oriented output; without --semester, defaults to the current semester",
			Hidden:  hidden,
			Local:   true,
		},
		&cli.StringFlag{
			Name:    "semester",
			Aliases: []string{"s"},
			Usage:   "Select semester(s) without prompts: current, all, index, 2025-1, 2025-2026, or comma-separated combinations",
			Hidden:  hidden,
			Local:   true,
		},
		&cli.StringFlag{
			Name:    "export",
			Aliases: []string{"e"},
			Usage:   "Export output to Desktop by default, or to a provided directory/file path",
			Hidden:  hidden,
			Local:   true,
		},
	}
}

func runGPACommand(c *cli.Command) error {
	opts, err := parseGPACommandOptions(c)
	if err != nil {
		return err
	}
	if opts.showHumanChrome() {
		printBanner(version)
	} else {
		skipUpdateCheck = true
	}
	runGPA(opts)
	return nil
}

func ensureLogin(verbose bool) (*api.API, error) {
	// Check if already logged in
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg != nil && cfg.HasCredentials() {
		if verbose {
			printInfo(fmt.Sprintf("Using saved credentials for: %s", cyan(cfg.Username)))
		}

		// Create HTTP client
		httpClient, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}

		apiClient := api.New(httpClient)

		// Try to login with saved credentials
		if err := performLoginWithHash(apiClient, cfg.Username, cfg.PasswordHash, verbose); err != nil {
			if verbose {
				printWarning("Saved credentials failed, please login again")
			}
			config.Delete()
			return nil, fmt.Errorf("authentication failed")
		}

		return apiClient, nil
	}

	return nil, fmt.Errorf("not logged in")
}

func runGPA(opts gpaCommandOptions) {
	apiClient, err := ensureLogin(!opts.suppressProgress())
	if err != nil {
		printError("You need to login first")
		fmt.Println()
		printInfo("Run: myxb login")
		os.Exit(1)
	}

	if !opts.suppressProgress() {
		printSuccess("Authentication successful!")
		fmt.Println()
	}

	calculateGPA(apiClient, opts)
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
	cfg, err := config.Load()
	if err != nil {
		printWarning(fmt.Sprintf("Failed to load existing config, recreating it: %v", err))
		cfg = nil
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.Username = username
	cfg.PasswordHash = config.HashPassword(password)

	if err := config.Save(cfg); err != nil {
		printWarning(fmt.Sprintf("Failed to save credentials: %v", err))
	} else {
		printSuccess("Credentials saved!")
		configPath, _ := config.GetConfigPath()
		fmt.Println(gray(fmt.Sprintf(" - Config saved to: %s", configPath)))
	}

	fmt.Println()
	if _, ok := cfg.ConfiguredScheduleProfile(); !ok {
		printInfo("Timetable commands default to the Standard profile")
		printInfo("Run 'myxb schedule profile highschool' to switch to the high-school bell schedule")
		fmt.Println()
	}
	printInfo("You can now run 'myxb' to calculate your GPA")
}

func runLogout() {
	// Check if there are saved credentials
	cfg, err := config.Load()
	if err != nil || cfg == nil || !cfg.HasCredentials() {
		printWarning("No saved credentials found")
		os.Exit(0)
	}

	// Delete the config file
	if err := config.Delete(); err != nil {
		printError(fmt.Sprintf("Failed to delete credentials: %v", err))
		os.Exit(1)
	}

	printSuccess("Credentials cleared!")
	if remainingCfg, err := config.Load(); err == nil && remainingCfg != nil && remainingCfg.ScheduleProfile != "" {
		fmt.Println(gray(fmt.Sprintf(" - Schedule profile kept: %s", remainingCfg.EffectiveScheduleProfile())))
	} else {
		configPath, _ := config.GetConfigPath()
		fmt.Println(gray(fmt.Sprintf(" - Config file removed: %s", configPath)))
	}
	fmt.Println()
	printInfo("Run 'myxb login' to login again")
}
