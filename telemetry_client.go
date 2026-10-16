package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type InfraiClient struct {
	Key  string
	HTTP *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c InfraiClient) Count(ctx context.Context, messages []map[string]string) (int, error) {
	var out struct {
		Tokens int `json:"tokens"`
	}
	err := c.post(ctx, "/ai/tokens/count", map[string]any{
		"model": "auto", "messages": messages,
	}, &out)
	return out.Tokens, err
}

func (c InfraiClient) Capture(ctx context.Context, job StreamJob, cause error) error {
	return c.post(ctx, "/errors/capture", map[string]any{
		"title":           "media creator delivery held",
		"message":         cause.Error(),
		"exception":       map[string]string{"message": cause.Error()},
		"level":           "error",
		"tags":            map[string]string{"asset_id": job.AssetID, "creator": job.Creator},
		"idempotency_key": job.AssetID,
	}, nil)
}

func (c InfraiClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
				continue
			}
			if env.Error != nil {
				return fmt.Errorf("Infrai %s: %s", env.Error.Code, env.Error.Message)
			}
			return fmt.Errorf("Infrai rejected the request")
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai request returned status %d", res.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request retry limit reached")
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
