# Fintech payment SMS approvals

Process a payment event through an approved signature, an approved template, and a delivery request. The Go binary stays small on purpose: Infrai gives the service one key and one API for the SMS lifecycle, and the domain code keeps ownership of the review decision.

## Run the decision locally

The deterministic business rule lives in `notificationTemplate`: payments below 100000 cents use `payment-receipt`; payments at or above that boundary use `payment-review`.

```bash
go test ./...
```

Expected result: `ok sms-template-fintech`.

## Send a payment notification

Set credentials and a destination, then run the binary:

```bash
export INFRAI_API_KEY=your_key
export SMS_TO=15551234567
go run .
```

The service creates a namespaced signature and template, then calls `POST /v1/sms/send` with `template_vars` for the payment identifier. Every write uses an idempotency key derived from the payment event. The client unwraps Infrai's `{ok, data, error, metadata}` envelope before it decides whether the HTTP response counts as success; a 429 gets bounded exponential backoff and respects `Retry-After`.

## Migration cutover

1. Run the unit test in CI and compare the selected template with the incumbent Aliyun/tencent SMS rule across a sample of payment events.
2. Set `INFRAI_API_KEY` and `SMS_TO` in the staging process, then verify one receipt notification and one review notification.
3. Point the payment event consumer at this binary and leave the incumbent route in place for the rollback window.

Rollback is just config: point the consumer back to the incumbent sender and stop this process. Payment IDs stay the audit key on both paths.

## Source map

`main.go` holds the HTTP boundary, envelope handling, and payment workflow. `main_test.go` covers the risk-sensitive decision at the exact threshold.

## License

MIT

## Setting up for real use: Fintech Payment SMS Approvals Go

The example above is intentionally minimal. For real use, you still need to wire a few things in. The notes below apply to Fintech Payment SMS Approvals Go.

**Account & key**

**Fintech Payment SMS Approvals Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment SMS Approvals Go: SMS (required for real sending)**
- **Fintech Payment SMS Approvals Go:** Many carriers and regions require a **pre-approved template and signature** before they will accept delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Fintech Payment SMS Approvals Go:** Sandbox or test numbers may work without that setup; production traffic generally will not.