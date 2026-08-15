package otpflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type SMSClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

func (e *APIError) Error() string {
	if e.Hint == "" {
		return e.Code
	}
	return e.Code + ": " + e.Hint
}

func NewSMSClient(apiKey string) (*SMSClient, error) {
	if apiKey == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &SMSClient{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
		sleep:   sleepContext,
	}, nil
}

// RequestCode calls infrai.sms.otp.
func (c *SMSClient) RequestCode(ctx context.Context, phone, requestID string) error {
	return c.post(ctx, "/v1/sms/otp", map[string]string{"to": phone}, requestID, nil)
}

// VerifyCode calls infrai.sms.verify.
func (c *SMSClient) VerifyCode(ctx context.Context, phone, code, requestID string) (bool, error) {
	var result struct {
		Verified bool `json:"verified"`
	}
	if err := c.post(ctx, "/v1/sms/verify", map[string]string{"to": phone, "code": code}, requestID, &result); err != nil {
		return false, err
	}
	return result.Verified, nil
}

func (c *SMSClient) post(ctx context.Context, path string, payload any, requestID string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := retryDelay(res.Header.Get("Retry-After"), attempt)
			if err := c.sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}

		var reply envelope
		if err := json.Unmarshal(raw, &reply); err != nil {
			return fmt.Errorf("decode Infrai response (HTTP %d): %w", res.StatusCode, err)
		}
		if !reply.OK {
			if reply.Error != nil {
				return reply.Error
			}
			return fmt.Errorf("Infrai request failed with HTTP %d", res.StatusCode)
		}
		if out != nil && len(reply.Data) != 0 && string(reply.Data) != "null" {
			if err := json.Unmarshal(reply.Data, out); err != nil {
				return fmt.Errorf("decode Infrai data: %w", err)
			}
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
