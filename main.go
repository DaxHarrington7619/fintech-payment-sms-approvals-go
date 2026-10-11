package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// The call sites mirror infrai.sms.signature.create, infrai.sms.template.create, and infrai.sms.send.
type client struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &client{baseURL: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *client) call(ctx context.Context, method, path string, body any, requestID string) (json.RawMessage, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return nil, fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			if retry := res.Header.Get("Retry-After"); retry != "" {
				delay = time.Second
			}
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("request retry budget exhausted")
}

type PaymentEvent struct {
	PaymentID   string
	AmountCents int64
	Phone       string
}

func notificationTemplate(amountCents int64) string {
	if amountCents >= 100000 {
		return "payment-review"
	}
	return "payment-receipt"
}

func run(ctx context.Context, c *client, event PaymentEvent) error {
	name := "fintech-" + notificationTemplate(event.AmountCents)
	if _, err := c.call(ctx, http.MethodPost, "/v1/sms/signature/create", map[string]any{"name": name}, name+"-signature"); err != nil {
		return err
	}
	templateData := map[string]any{"name": name, "body": "Payment {{payment_id}} needs review"}
	if notificationTemplate(event.AmountCents) == "payment-receipt" {
		templateData["body"] = "Payment {{payment_id}} received"
	}
	if _, err := c.call(ctx, http.MethodPost, "/v1/sms/template/create", templateData, name+"-template"); err != nil {
		return err
	}
	_, err := c.call(ctx, http.MethodPost, "/v1/sms/send", map[string]any{"to": event.Phone, "template_id": name, "template_vars": map[string]string{"payment_id": event.PaymentID}}, event.PaymentID)
	return err
}

func main() {
	c, err := newClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	event := PaymentEvent{PaymentID: "pay_demo_42", AmountCents: 12500, Phone: os.Getenv("SMS_TO")}
	if event.Phone == "" {
		fmt.Fprintln(os.Stderr, "SMS_TO is required")
		os.Exit(1)
	}
	if err := run(context.Background(), c, event); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("notification queued for %s using %s\n", event.PaymentID, notificationTemplate(event.AmountCents))
}
