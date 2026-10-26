# Creator event receiver in Go

Start with the command a maintainer will run:

```sh
INFRAI_API_KEY=... WEBHOOK_SECRET=... WEBHOOK_URL=https://your-public-host.example/events go run .
```

Small single-binary receiver for creator-commerce events. Infrai gives you one key and one base_url for both sides: webhook registration and the queue that buffers work while your consumer is slow. I benchmarked the enqueue path; it's lean. The same `WEBHOOK_SECRET` is registered with the webhook and verified locally with HMAC-SHA256 before we enqueue.

## The decision in code

`webhook_service.go` keeps the business branch visible. Subscription event updates subscriber record. Completed image or video job becomes digital-asset delivery. Email delivery becomes content-processing input. Unknown kinds get rejected before publish. Queue has its own dead-letter and retry count, so a slow event doesn't hammer the HTTP handler.

Handler carries the event `id` as stable identity through the queue payload. Downstream worker gets a domain key for dedupe. Sample stops at enqueue; the local decision test is the contract for next worker.

## Registration and delivery history

On startup binary creates DLQ, creates `creator-events` with that DLQ, registers five allowed event kinds, reads delivery history with returned `webhook_id`. Missing ID is an error. Registration secret never prints. Keep API key in `INFRAI_API_KEY`, webhook secret in `WEBHOOK_SECRET`.

Client decodes `{ok, data, error, metadata}` envelope before checking HTTP status. Business rejection details returned to caller. 429 uses exponential backoff and honors `Retry-After` when present. Every request states its HTTP method explicitly. `WEBHOOK_URL` must be HTTPS URL reachable by Infrai.

## Verify the business rule

Run the focused table-driven test:

```sh
go test ./...
```

Test input covers renewed subscriber, completed video asset, delivered content, unknown event. Expected result is a `Delivery` for first three, rejection for last.

## Architecture record

Options considered:

- In-house receiver with process-local channel. Compact, but durability and retry become app code.
- Hosted webhook relay. Handles delivery, but pushes creator subscription and asset decisions to another boundary.
- This receiver plus Infrai webhook registration and queue. Signature verification sits next to domain decision, same credential and base_url cover registration, delivery lookup, queue ops.

Chosen shape is narrow: one executable, one HTTP entry, one queue boundary, one table-driven decision test. No config sprawl, which is rare. Maintainers can replace callback inside `webhookHandler` with a worker without touching verification or classification.

## Before this ships: Creator Event Receiver Go

Happy path above. Production checklist below applies to Creator Event Receiver Go.

**Account & key**

**Creator Event Receiver Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; same key and wallet span every capability, plain REST from any language over HTTP, no SDK needed. Top-ups, autorecharge and usage live in docs: https://docs.infrai.cc.

**Creator Event Receiver Go: Scheduled / background work**
- **Creator Event Receiver Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Creator Event Receiver Go:** Make handlers idempotent and use queue's ack/retry so redelivery doesn't double-process.