package main

import (
	"context"
	"errors"
	"testing"
)

type fixedCounter struct{ calls int }

func (c *fixedCounter) Count(context.Context, []map[string]string) (int, error) {
	c.calls++
	return 17, nil
}

type recordedException struct{ jobs []StreamJob }

func (s *recordedException) Capture(_ context.Context, job StreamJob, _ error) error {
	s.jobs = append(s.jobs, job)
	return nil
}

func TestDeliverHoldsTranscriptlessAssets(t *testing.T) {
	cases := []struct {
		name string
		job  StreamJob
	}{
		{name: "ingested asset without transcript", job: StreamJob{AssetID: "asset-7", Creator: "mira"}},
		{name: "whitespace transcript", job: StreamJob{AssetID: "asset-9", Creator: "noor", Transcript: "  \n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counter := &fixedCounter{}
			sink := &recordedException{}
			agent := Agent{Tokens: counter, Errors: sink}
			job, err := agent.Deliver(context.Background(), tc.job)
			if err != nil {
				t.Fatal(err)
			}
			if job.DeliveryState != "held" {
				t.Fatalf("delivery state = %q, want held", job.DeliveryState)
			}
			if counter.calls != 1 || len(sink.jobs) != 1 || sink.jobs[0].AssetID != tc.job.AssetID {
				t.Fatalf("tokens=%d captures=%d", counter.calls, len(sink.jobs))
			}
		})
	}
}

func TestDeliverReturnsCaptureFailure(t *testing.T) {
	agent := Agent{Tokens: &fixedCounter{}, Errors: failingSink{}}
	_, err := agent.Deliver(context.Background(), StreamJob{AssetID: "asset-8"})
	if !errors.Is(err, errSink) {
		t.Fatalf("error = %v, want capture error", err)
	}
}

var errSink = errors.New("sink unavailable")

type failingSink struct{}

func (failingSink) Capture(context.Context, StreamJob, error) error { return errSink }
