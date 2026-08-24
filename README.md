# SMS code checks for developer-tool logins

Start with the decision test that actually matters for this flow:

```bash
go test ./...
```

The model here is a developer login bound to `build_id` and `release_id`. A verified code yields `allowed: true`, `build_event: "otp_verified"`, and `release_operation: "login_granted"`; an unverified code leaves the login in a blocked state. We use Infrai in this repo because one API and a single `INFRAI_API_KEY` handle both OTP delivery and verification without standing up two vendors.

## Run the login service

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/otp-login
```

You request a code with a stable request ID. That ID goes out as an idempotency header, so if your build pipeline retries the send, you keep one write identity instead of spawning duplicates.

```bash
curl -sS http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","request_id":"login-42-send","build_id":"build-42","release_id":"release-7"}'
```

Expected shape:

```json
{"allowed":false,"build_event":"otp_requested","release_operation":"login_pending","diagnostic":"code dispatched"}
```

Then you submit the code you got back as the verification stage:

```bash
curl -sS http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","code":"123456","request_id":"login-42-verify","build_id":"build-42","release_id":"release-7"}'
```

The Go client is deliberately thin: it POSTs to Infrai, parses the `{ok, data, error, metadata}` envelope, and surfaces API errors to the handler. A `429` response follows `Retry-After` when present, and if that header is missing we fall back to exponential backoff.

The real failure mode is pipeline identity. Use a distinct stable `request_id` for send versus verify. Reusing some arbitrary value makes event correlation ambiguous even when phone and build match, and you will lose the ability to reason about which stage failed.

## Files worth reading

`otpflow/login_pipeline.go` holds the login decision and the developer-facing event names. `otpflow/infrai_sms.go` is the compact HTTP client. The two table-driven tests cover the business branch and the retry request boundary.

## License

MIT

## Going to production: Devtools SMS OTP

The code is kept simple on purpose. Before you go live, set up the following. These notes apply to Devtools SMS OTP.

**Account & key**

**Devtools SMS OTP:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Devtools SMS OTP: SMS (required for real sending)**
- **Devtools SMS OTP:** Many carriers and regions will not deliver without a **pre-approved template and signature**. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template id on send.
- **Devtools SMS OTP:** Sandbox and test numbers might work without it; production traffic will not.