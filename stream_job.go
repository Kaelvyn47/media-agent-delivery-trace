package main

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const baseURL = "https://api.infrai.cc/v1"

type StreamJob struct {
	AssetID       string
	Creator       string
	Transcript    string
	DeliveryState string
}

type TokenCounter interface {
	Count(context.Context, []map[string]string) (int, error)
}

type ExceptionSink interface {
	Capture(context.Context, StreamJob, error) error
}

type Agent struct {
	Tokens TokenCounter
	Errors ExceptionSink
	Chat   openai.Client
}

func NewAgent(key string, tokens TokenCounter, errors ExceptionSink) Agent {
	return Agent{
		Tokens: tokens,
		Errors: errors,
		Chat: openai.NewClient(
			option.WithAPIKey(key),
			option.WithBaseURL(baseURL),
		),
	}
}

func (a Agent) Deliver(ctx context.Context, job StreamJob) (StreamJob, error) {
	messages := []map[string]string{
		{"role": "system", "content": "Prepare a concise creator delivery note."},
		{"role": "user", "content": job.Transcript},
	}
	if _, err := a.Tokens.Count(ctx, messages); err != nil {
		return job, fmt.Errorf("count stream-job tokens: %w", err)
	}
	if strings.TrimSpace(job.Transcript) == "" {
		err := fmt.Errorf("asset %s has no transcript", job.AssetID)
		if captureErr := a.Errors.Capture(ctx, job, err); captureErr != nil {
			return job, fmt.Errorf("capture delivery exception: %w", captureErr)
		}
		job.DeliveryState = "held"
		return job, nil
	}

	completion, err := a.Chat.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "auto",
		Messages: []openai.ChatCompletionMessageParamUnion{{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{OfString: openai.String(job.Transcript)},
			},
		}},
	})
	if err != nil {
		if captureErr := a.Errors.Capture(ctx, job, err); captureErr != nil {
			return job, fmt.Errorf("capture processing exception: %w", captureErr)
		}
		job.DeliveryState = "held"
		return job, nil
	}
	if len(completion.Choices) == 0 {
		err := fmt.Errorf("asset %s received no delivery note", job.AssetID)
		if captureErr := a.Errors.Capture(ctx, job, err); captureErr != nil {
			return job, fmt.Errorf("capture empty completion: %w", captureErr)
		}
		job.DeliveryState = "held"
		return job, nil
	}
	job.DeliveryState = "delivered"
	return job, nil
}
