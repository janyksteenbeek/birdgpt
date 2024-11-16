package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/janyksteenbeek/birdgpt/config"
	"github.com/janyksteenbeek/birdgpt/internal/gmail"
)

type EmailProcessor struct {
	cfg        *config.Config
	gmail      *gmail.Client
	lastUpdate time.Time
}

func NewEmailProcessor(cfg *config.Config, gmailClient *gmail.Client) *EmailProcessor {
	lastUpdate, err := time.Parse(time.RFC3339, cfg.App.LastUpdate)
	if err != nil {
		log.Fatal("Error parsing last update time: %v", err)
	}

	return &EmailProcessor{
		cfg:        cfg,
		gmail:      gmailClient,
		lastUpdate: lastUpdate,
	}
}

func (p *EmailProcessor) ProcessEmails(ctx context.Context) ([]gmail.Email, error) {
	log.Printf("Checking for new emails since %v...", p.lastUpdate.Format(time.RFC3339))
	emails, err := p.gmail.FetchEmails(ctx, p.cfg.Gmail.SearchLabel, p.lastUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	if len(emails) == 0 {
		log.Println("No new emails found")
		return nil, nil
	}

	log.Printf("Found %d new emails", len(emails))
	return emails, nil
}

func (p *EmailProcessor) UpdateLastProcessed() error {
   p.lastUpdate = time.Now()
   p.cfg.App.LastUpdate = p.lastUpdate.Format(time.RFC3339)
   
   if err := config.SaveConfig(p.cfg); err != nil {
	   return fmt.Errorf("failed to save config: %w", err)
   }
   
   log.Printf("Updated last processed time to: %v", p.lastUpdate.Format(time.RFC3339))
   return nil
} 