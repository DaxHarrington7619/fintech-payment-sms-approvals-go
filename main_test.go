package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotificationTemplateRiskSplit(t *testing.T) {
	cases := []struct {
		amount int64
		want   string
	}{{99999, "payment-receipt"}, {100000, "payment-review"}}
	for _, tc := range cases {
		if got := notificationTemplate(tc.amount); got != tc.want {
			t.Fatalf("amount %d: got %q, want %q", tc.amount, got, tc.want)
		}
	}
}

func TestRunRequestFields(t *testing.T) {
	for _, tc := range []struct {
		amount int64
		body   string
	}{{12500, "Payment {{payment_id}} received"}, {100000, "Payment {{payment_id}} needs review"}} {
		t.Run(tc.body, func(t *testing.T) {
			requests := []struct {
				path string
				body map[string]any
			}{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				requests = append(requests, struct {
					path string
					body map[string]any
				}{r.URL.Path, body})
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
			}))
			defer server.Close()
			c := &client{baseURL: server.URL, key: "test", http: server.Client()}
			if err := run(context.Background(), c, PaymentEvent{PaymentID: "pay_test", AmountCents: tc.amount, Phone: "test-only"}); err != nil {
				t.Fatal(err)
			}
			if len(requests) != 3 {
				t.Fatalf("got %d requests, want 3", len(requests))
			}
			if requests[0].path != "/v1/sms/signature/create" || len(requests[0].body) != 1 || requests[0].body["name"] == nil {
				t.Errorf("invalid signature request: %+v", requests[0])
			}
			if requests[1].path != "/v1/sms/template/create" || len(requests[1].body) != 2 || requests[1].body["name"] == nil || requests[1].body["body"] != tc.body {
				t.Errorf("invalid template request: %+v", requests[1])
			}
			if requests[2].path != "/v1/sms/send" || requests[2].body["to"] != "test-only" {
				t.Errorf("invalid send request: %+v", requests[2])
			}
		})
	}
}
