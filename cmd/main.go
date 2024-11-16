package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/janyksteenbeek/birdgpt/config"
	"github.com/janyksteenbeek/birdgpt/internal/gmail"
	"github.com/janyksteenbeek/birdgpt/internal/moneybird"
	"github.com/janyksteenbeek/birdgpt/internal/openai"
	"github.com/janyksteenbeek/birdgpt/internal/processor"
)

func main() {
	log.Println("[github.com/janyksteenbeek/birdgpt]")
	log.Println("Starting invoice processor...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupGracefulShutdown(cancel)

	log.Println("Initializing clients...")
	gmailClient, err := initializeGmail(ctx, cfg)
	if err != nil {
		log.Fatalf("Gmail initialization failed: %v", err)
	}

	moneybirdClient, err := moneybird.NewClient(cfg.Moneybird.Token, cfg.Moneybird.AdminID)
	if err != nil {
		log.Fatalf("Moneybird initialization failed: %v", err)
	}

	openaiClient := openai.NewClient(cfg.OpenAI.APIKey)

	log.Println("Testing connections...")
	if err := testConnections(ctx, cfg, gmailClient, moneybirdClient); err != nil {
		log.Fatalf("Connection test failed: %v", err)
	}

	proc := processor.New(cfg, gmailClient, moneybirdClient, openaiClient)
	log.Printf("Invoice processor started. Checking for new emails every %v...", cfg.App.SleepTime)
	if err := proc.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Processor error: %v", err)
	}
}

func testConnections(ctx context.Context, cfg *config.Config, gmail *gmail.Client, moneybird *moneybird.Client) error {
	if _, err := gmail.FetchEmails(ctx, cfg.Gmail.SearchLabel, time.Now().Add(-time.Minute)); err != nil {
		return fmt.Errorf("gmail test failed: %w", err)
	}

	if _, err := moneybird.SearchContacts("test"); err != nil {
		return fmt.Errorf("moneybird test failed: %w", err)
	}

	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Cya later!")
		cancel()
	}()
}

func initializeGmail(ctx context.Context, cfg *config.Config) (*gmail.Client, error) {
	client, authURL, err := gmail.Setup(ctx, cfg.Gmail.CredentialsFile, cfg.Gmail.Token)
	if err != nil {
		return nil, fmt.Errorf("gmail setup failed: %w", err)
	}

	if authURL != "" {
		fmt.Printf("Visit this URL to authorize Gmail access:\n%s\n", authURL)
		fmt.Print("Enter the authorization code: ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()

		token, err := gmail.Exchange(ctx, cfg.Gmail.CredentialsFile, scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("token exchange failed: %w", err)
		}

		cfg.Gmail.Token = token
		if err := config.SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("saving config: %w", err)
		}

		return initializeGmail(ctx, cfg)
	}

	return client, nil
}
