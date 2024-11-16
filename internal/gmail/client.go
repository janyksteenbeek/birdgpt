package gmail

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "time"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/gmail/v1"
    "google.golang.org/api/option"
)

type Client struct {
    service *gmail.Service
}

type Email struct {
    ID          string
    From        string
    Subject     string
    Body        string
    Date        time.Time
    Attachments [][]byte
}

func Setup(ctx context.Context, credentialsFile string, token string) (*Client, string, error) {
    creds, err := os.ReadFile(credentialsFile)
    if err != nil {
        return nil, "", fmt.Errorf("reading credentials: %w", err)
    }

    config, err := google.ConfigFromJSON(creds, gmail.GmailReadonlyScope)
    if err != nil {
        return nil, "", fmt.Errorf("parsing credentials: %w", err)
    }

    if token == "" {
        return nil, config.AuthCodeURL("state"), nil
    }

    oauthClient := config.Client(ctx, &oauth2.Token{AccessToken: token})
    srv, err := gmail.NewService(ctx, option.WithHTTPClient(oauthClient))
    if err != nil {
        return nil, "", fmt.Errorf("creating gmail service: %w", err)
    }

    return &Client{service: srv}, "", nil
}

func Exchange(ctx context.Context, credentialsFile string, code string) (string, error) {
    creds, err := os.ReadFile(credentialsFile)
    if err != nil {
        return "", fmt.Errorf("reading credentials: %w", err)
    }

    config, err := google.ConfigFromJSON(creds, gmail.GmailReadonlyScope)
    if err != nil {
        return "", fmt.Errorf("parsing credentials: %w", err)
    }

    token, err := config.Exchange(ctx, code)
    if err != nil {
        return "", fmt.Errorf("exchanging code: %w", err)
    }

    return token.AccessToken, nil
}

func (c *Client) FetchEmails(ctx context.Context, label string, after time.Time) ([]Email, error) {
    query := fmt.Sprintf("label:%s after:%d", label, after.Unix())
    msgs, err := c.service.Users.Messages.List("me").Q(query).Do()
    if err != nil {
        return nil, fmt.Errorf("listing messages: %w", err)
    }

    var emails []Email
    for _, msg := range msgs.Messages {
        email, err := c.fetchEmail(msg.Id)
        if err != nil {
            return nil, fmt.Errorf("fetching email %s: %w", msg.Id, err)
        }
        emails = append(emails, *email)
    }

    return emails, nil
}

func (c *Client) fetchEmail(messageID string) (*Email, error) {
    msg, err := c.service.Users.Messages.Get("me", messageID).Do()
    if err != nil {
        return nil, err
    }

    email := &Email{ID: messageID}
    for _, header := range msg.Payload.Headers {
        switch header.Name {
        case "From":
            email.From = header.Value
        case "Subject":
            email.Subject = header.Value
        case "Date":
            if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
                email.Date = t
            }
        }
    }

    email.Body, email.Attachments = c.extractContent(messageID, msg.Payload)
    return email, nil
}

func (c *Client) extractContent(messageID string, part *gmail.MessagePart) (string, [][]byte) {
    var body string
    var attachments [][]byte

    if part.Body != nil && part.Body.Data != "" {
        if data, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
            body = string(data)
        }
    }

    for _, p := range part.Parts {
        if p.Body != nil && p.Body.Data != "" {
            if data, err := base64.URLEncoding.DecodeString(p.Body.Data); err == nil {
                body = string(data)
            }
            continue
        }

        if p.Body != nil && p.Body.AttachmentId != "" {
            att, err := c.service.Users.Messages.Attachments.Get("me", messageID, p.Body.AttachmentId).Do()
            if err == nil {
                if data, err := base64.URLEncoding.DecodeString(att.Data); err == nil {
                    attachments = append(attachments, data)
                }
            }
        }
    }

    return body, attachments
} 