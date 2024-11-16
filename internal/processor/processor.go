package processor

import (
	"context"
	"log"
	"time"

	"github.com/janyksteenbeek/birdgpt/config"
	"github.com/janyksteenbeek/birdgpt/internal/gmail"
	"github.com/janyksteenbeek/birdgpt/internal/moneybird"
	"github.com/janyksteenbeek/birdgpt/internal/openai"
)

type Processor struct {
	emailProcessor     *EmailProcessor
	invoiceProcessor   *InvoiceProcessor
	moneybirdProcessor *MoneybirdProcessor
	cfg               *config.Config
}

func New(cfg *config.Config, gmailClient *gmail.Client, moneybirdClient *moneybird.Client, openaiClient *openai.Client) *Processor {
	return &Processor{
		emailProcessor:     NewEmailProcessor(cfg, gmailClient),
		invoiceProcessor:   NewInvoiceProcessor(openaiClient),
		moneybirdProcessor: NewMoneybirdProcessor(cfg, moneybirdClient),
		cfg:               cfg,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	if err := p.processNewEmails(ctx); err != nil {
		log.Printf("Error processing emails: %v", err)
	}

	ticker := time.NewTicker(p.cfg.App.SleepTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.processNewEmails(ctx); err != nil {
				log.Printf("Error processing emails: %v", err)
			}
		}
	}
}

func (p *Processor) processNewEmails(ctx context.Context) error {
	emails, err := p.emailProcessor.ProcessEmails(ctx)
	if err != nil {
		return err
	}

	for _, email := range emails {
		invoiceData, err := p.invoiceProcessor.ProcessEmail(ctx, email)
		if err != nil {
			log.Printf("Failed to process email %s: %v", email.Subject, err)
			continue
		}

		if invoiceData != nil {
			if err := p.moneybirdProcessor.ProcessInvoice(ctx, invoiceData); err != nil {
				log.Printf("Failed to process invoice for email %s: %v", email.Subject, err)
				continue
			}
		}

		if err := p.emailProcessor.UpdateLastProcessed(email); err != nil {
			log.Printf("Failed to update last processed time: %v", err)
		}
	}

	return nil
}
