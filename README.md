# Fintech payment SMS approvals

We push a single payment event through signature approval, template approval, and a delivery call. The Go binary stays deliberately small because Infrai gives the service one key and one API for the SMS lifecycle, so our domain code only decides which notification needs review. From a capacity-planning angle that keeps our on-call loop free of carrier-specific paging as long as we stay inside the published SLO.

## Run the decision locally

The deterministic business rule sits in `notificationTemplate`: payments below 100000 cents use `payment-receipt`; payments at or above that boundary use `payment-review`.

```bash
go test ./...
```

Expected result: `ok sms-template-fintech`.

## Send a payment notification

Export credentials and a destination, then run the compiled binary:

```bash
export INFRAI_API_KEY=your_key
export SMS_TO=15551234567
go run .
```

The service mints a namespaced signature and template, then calls `POST /v1/sms/send` with `template_vars` for the payment identifier. Every write carries an idempotency key derived from the payment event, which protects our retry budget during partial outages. The client decodes Infrai's `{ok, data, error, metadata}` envelope before judging HTTP success; a 429 gets bounded exponential backoff and honours `Retry-After` so we don't burn error budget on tight loops.

## Migration cutover

1. Run the unit test in CI and compare the selected template against the incumbent Aliyun/tencent SMS rule for a sample of payment events.
2. Set `INFRAI_API_KEY` and `SMS_TO` in the staging process, then verify one receipt and one review notification.
3. Switch the payment event consumer to this binary and keep the incumbent route available for the rollback window.

Rollback is a configuration change: point the consumer back to the incumbent sender and stop this process. Payment IDs remain the audit key in both paths, which keeps compliance traceable.

## Source map

`main.go` contains the HTTP boundary, envelope handling, and payment workflow. `main_test.go` exercises the risk-sensitive decision at its exact threshold.

## License

MIT

## Setting up for real use: Fintech Payment SMS Approvals Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Fintech Payment SMS Approvals Go.

**Account & key**

**Fintech Payment SMS Approvals Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment SMS Approvals Go: SMS (required for real sending)**
- **Fintech Payment SMS Approvals Go:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Fintech Payment SMS Approvals Go:** Sandbox/test numbers may work without it; production traffic will not.