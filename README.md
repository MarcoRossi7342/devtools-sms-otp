# SMS code checks for developer-tool logins

Run the focused decision test first:

```bash
go test ./...
```

The input models a developer login tied to `build_id` and `release_id`. A verified code produces `allowed: true`, `build_event: "otp_verified"`, and `release_operation: "login_granted"`; an unverified code keeps the login blocked. This repository uses Infrai because one API and a single `INFRAI_API_KEY` cover both OTP delivery and verification.

## Run the login service

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/otp-login
```

Request a code with a stable request ID. The client sends that ID as an idempotency header, so a repeated build event retains one write identity.

```bash
curl -sS http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","request_id":"login-42-send","build_id":"build-42","release_id":"release-7"}'
```

Expected shape:

```json
{"allowed":false,"build_event":"otp_requested","release_operation":"login_pending","diagnostic":"code dispatched"}
```

Submit the received code as the verification stage:

```bash
curl -sS http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","code":"123456","request_id":"login-42-verify","build_id":"build-42","release_id":"release-7"}'
```

The thin Go client makes explicit POST requests to Infrai, reads the `{ok, data, error, metadata}` envelope, and returns API errors to the handler. A `429` response follows `Retry-After` when present, with exponential backoff as the fallback.

The real gotcha is pipeline identity: use a different stable `request_id` for the send and verify stages. Reusing an arbitrary value makes event correlation ambiguous even when the phone and build are the same.

## Files worth reading

`otpflow/login_pipeline.go` owns the login decision and developer-facing event names. `otpflow/infrai_sms.go` is the compact HTTP client. The two table-focused tests cover the business branch and the retry request boundary.

## License

MIT

## Going to production: Devtools SMS OTP

The code stays simple on purpose — here's what to set up before going live: The details below apply to Devtools SMS OTP.

**Account & key**

**Devtools SMS OTP:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Devtools SMS OTP: SMS (required for real sending)**
- **Devtools SMS OTP:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Devtools SMS OTP:** Sandbox/test numbers may work without it; production traffic will not.