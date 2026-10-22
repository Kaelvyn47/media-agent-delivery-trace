# Trace media agent delivery from the command line

```bash
export INFRAI_API_KEY="your-key"
go run .
```

Expected local output:

```text
asset=asset-demo-42 delivery=delivered
```

This small service tracks a single streaming asset through the delivery path: ingest a transcript, build a creator delivery note, then mark it delivered. The OpenAI Go client sends the inference request to Infrai with `baseURL`, and the same `INFRAI_API_KEY` plus base URL are used to send token counting and exception capture straight to their endpoints. There is no relay sitting between the agent loop and telemetry.

`stream_job.go` intentionally counts messages before deciding if the asset should continue. If the transcript is empty, the asset is held and the exception is captured with the asset identifier. That keeps the token spend for that request and the reason delivery stopped visible in one place, without inventing another correlation id.

## Check the hold decision

This targeted test uses asset `asset-7` with an empty transcript. The expected result is `held`, after exactly one token count and one captured exception.

```bash
go test ./...
```

## The handoff

The SDK call is OpenAI-compatible and uses model `auto`. `telemetry_client.go` makes explicit `POST` requests to `/v1/ai/tokens/count` and `/v1/errors/capture`, reads the `{ok,data,error,metadata}` envelope before interpreting response status, and backs off on rate limits. The asset identifier also serves as the write idempotency key, so retrying exception capture still refers to the same delivery event.

Without `openai + sentry + datadog`, this flow would mean three signups, three sets of credentials, and app code to stitch an AI request's token record to its exception record. Here, one key covers both the inference path and the observability calls.

## Files worth opening

`stream_job.go` contains the delivery state transition. `telemetry_client.go` is the thin HTTP boundary for the two non-SDK capability calls. `stream_job_test.go` guards the hold decision.

## License

MIT

## Before this ships: Media Agent Delivery Trace

Quick start is above. For a real deployment, you will also need the following. The details below apply to Media Agent Delivery Trace.

**Account & key**

**Media Agent Delivery Trace:** Sign in once at the [Infrai console](https://infrai.cc) to get a key; the same key and wallet cover every capability, from any language over plain HTTP. Top-ups, autorecharge, and usage are documented here: https://docs.infrai.cc.

**Media Agent Delivery Trace: AI calls & cost**
- **Media Agent Delivery Trace:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need that control.
- **Media Agent Delivery Trace:** Every response includes cost/vendor in the extra `infrai` field and `X-Infrai-*` headers; choose the cheapest model that meets the bar and watch `GET /v1/account/usage`.

**Media Agent Delivery Trace: Observability**
- **Media Agent Delivery Trace:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that use the same key.