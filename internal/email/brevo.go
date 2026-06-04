package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type Message struct {
	To    string
	Title string
	Body  string
}

type BrevoSender struct {
	APIKey      string
	SenderName  string
	SenderEmail string
	Client      *http.Client
}

func (s BrevoSender) Send(ctx context.Context, message Message) error {
	if s.APIKey == "" || s.SenderEmail == "" {
		return nil
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	payload := map[string]any{
		"to":          []map[string]string{{"email": message.To}},
		"subject":     message.Title,
		"htmlContent": message.Body,
		"sender":      map[string]string{"name": s.SenderName, "email": s.SenderEmail},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("brevo send failed")
	}
	return nil
}
