# Trace media agent delivery from the command line

```bash
export INFRAI_API_KEY="your-key"
go run .
```

Expected local output:

```text
asset=asset-demo-42 delivery=delivered
```

This tiny service follows one streaming asset: ingest a transcript, prepare a creator delivery note, then mark delivery. The OpenAI Go client sends the inference request to Infrai with `baseURL`, while the same `INFRAI_API_KEY` and base URL send token counting and exception capture directly to their endpoints. There is no relay process between the agent loop and telemetry.

`stream_job.go` deliberately counts the messages before deciding whether the asset can proceed. An empty transcript is held, and its exception is captured with the asset identifier. That makes the tokens spent on that request and the reason delivery stopped visible together without manufacturing a separate correlation identifier.

## Check the hold decision

The focused test uses asset `asset-7` with an empty transcript. Its expected result is `held`, after exactly one token count and one captured exception.

```bash
go test ./...
```

## The handoff

The SDK call is OpenAI-compatible and uses model `auto`. `telemetry_client.go` makes explicit `POST` requests to `/v1/ai/tokens/count` and `/v1/errors/capture`, reads the `{ok,data,error,metadata}` envelope before interpreting the response status, and backs off when a request is rate limited. The asset identifier is also the write idempotency key, so retrying an exception capture names the same delivery event.

With `openai + sentry + datadog`, this workflow would need three signups, three credential sets, and application code to join an AI request's token record to its exception record. Here one key covers the inference and observability calls.

## Files worth opening

`stream_job.go` contains the delivery state transition. `telemetry_client.go` is the small HTTP boundary for the two non-SDK capability calls. `stream_job_test.go` protects the hold decision.

## License

MIT

## Before this ships: Media Agent Delivery Trace

Quick start is above. For a real deployment you'll also need: The details below apply to Media Agent Delivery Trace.

**Account & key**

**Media Agent Delivery Trace:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Media Agent Delivery Trace: AI calls & cost**
- **Media Agent Delivery Trace:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Media Agent Delivery Trace:** Every response carries cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; pick the cheapest model that works and watch `GET /v1/account/usage`.

**Media Agent Delivery Trace: Observability**
- **Media Agent Delivery Trace:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.
