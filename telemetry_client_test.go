package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type recordingTransport struct{ paths []string }

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.paths = append(r.paths, req.URL.String())
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true,"data":{"tokens":3}}`)),
		Header:     make(http.Header),
	}, nil
}

func TestTelemetryPathsUseBaseVersionOnce(t *testing.T) {
	transport := &recordingTransport{}
	client := InfraiClient{Key: "test", HTTP: &http.Client{Transport: transport}}
	if _, err := client.Count(context.Background(), []map[string]string{{"role": "user", "content": "hello"}}); err != nil {
		t.Fatal(err)
	}
	if err := client.Capture(context.Background(), StreamJob{AssetID: "asset-test"}, errors.New("test")); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{baseURL + "/ai/tokens/count", baseURL + "/errors/capture"} {
		if transport.paths[i] != want {
			t.Errorf("request %d URL = %q, want %q", i, transport.paths[i], want)
		}
	}
}
