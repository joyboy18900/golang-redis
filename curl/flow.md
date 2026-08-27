# Manual test flow

Full walkthrough for exercising the API by hand, from starting the stack to
tearing it down.

## Start

```bash
docker compose up -d --build
docker compose ps
docker compose logs app1 app2 --tail 20   # should show "server started on port 8080" twice
```

## 1. Rate limiter - watch it count down and reset

```bash
for i in 1 2 3 4 5 6; do curl -s -D - -o /dev/null http://localhost:8080/ping | grep -i ratelimit; done
```

The first `rate_limit.limit` (default 5) requests return `200` with a
falling `X-RateLimit-Remaining`; the next one returns `429`:

```json
{ "code": 429, "message": "rate limit exceeded", "data": null }
```

Wait past `rate_limit.window_seconds` (default 10s) and it opens back up:

```bash
sleep 11
curl -s -w " [%{http_code}]\n" http://localhost:8080/ping
```

```json
{ "code": 200, "message": "pong", "data": null } [200]
```

## 2. Distributed lock - single call

```bash
curl -X POST http://localhost:8080/jobs/run \
  -H "Content-Type: application/json" \
  -d '{"job_id":"demo-job","hold_millis":800}'
```

```json
{ "code": 200, "message": "critical section completed", "data": { "job_id": "demo-job", "started_at": "...", "finished_at": "..." } }
```

## 3. Distributed lock - across two independent app instances

Fire the same `job_id` at both instances at once. Exactly one wins:

```bash
curl -s -w " [%{http_code}]\n" -X POST http://localhost:8080/jobs/run \
  -H "Content-Type: application/json" -d '{"job_id":"demo-job","hold_millis":800}' &
sleep 0.1
curl -s -w " [%{http_code}]\n" -X POST http://localhost:8081/jobs/run \
  -H "Content-Type: application/json" -d '{"job_id":"demo-job","hold_millis":800}'
```

```json
{ "code": 200, "message": "critical section completed", "data": { "job_id": "demo-job", "..." } } [200]
{ "code": 409, "message": "job demo-job is already running", "data": null } [409]
```

## 4. Pub/Sub - publish and watch fan-out

Both app instances run `pubsub.consumer_count` (default 2) background
consumers each, so a single publish reaches all of them:

```bash
curl -X POST http://localhost:8081/messages \
  -H "Content-Type: application/json" \
  -d '{"channel":"notifications","payload":"hello from flow.md"}'
```

```json
{ "code": 200, "message": "message published", "data": { "channel": "notifications", "payload": "hello from flow.md", "subscriber_count": 4 } }
```

```bash
docker compose logs app1 app2 | grep "received on"
```

```
app1-1  | ... consumer-1 received on notifications: hello from flow.md
app1-1  | ... consumer-2 received on notifications: hello from flow.md
app2-1  | ... consumer-1 received on notifications: hello from flow.md
app2-1  | ... consumer-2 received on notifications: hello from flow.md
```

## 5. Rejection cases worth checking

Missing `job_id`:

```bash
curl -X POST http://localhost:8080/jobs/run -H "Content-Type: application/json" -d '{}'
```

```json
{ "code": 422, "message": "job_id is required", "data": null }
```

`hold_millis` larger than the configured lock TTL (default 10s):

```bash
curl -X POST http://localhost:8080/jobs/run \
  -H "Content-Type: application/json" -d '{"job_id":"x","hold_millis":20000}'
```

```json
{ "code": 422, "message": "hold_millis must not exceed the lock ttl", "data": null }
```

Publishing with no payload:

```bash
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" -d '{"channel":"notifications"}'
```

```json
{ "code": 422, "message": "payload is required", "data": null }
```

## Stop

```bash
docker compose down
```
