# golang-redis

Redis patterns beyond a key-value cache: a Redsync-based distributed lock,
a Lua-scripted sliding-window rate limiter, and Pub/Sub fan-out.

## Run

```
docker-compose up --build
```

Starts one `redis` and two independent `app` instances (`:8080`, `:8081`)
so the lock demo can be exercised across separate processes, the way the
SPEC's "multiple app instances" requirement actually means.

## Endpoints

```
POST /jobs/run
GET  /ping
POST /messages
```

See `curl/flow.md` for full request/response examples, including firing
`/jobs/run` at both app instances at the same time.

- `/jobs/run`'s lock is a single-node `SET NX PX` via redsync, not full
  multi-node Redlock, and gives no fencing token. It suits best-effort
  de-duplication only; use a DB-level lock where a duplicate run would be
  a correctness bug.

## Tests

```
go test ./...
go generate ./...   # regenerate repository/service mocks
```

Integration tests (`*_integration_test.go` at the project root) need a
real Redis on `localhost:6379` and skip automatically if it is
unreachable. See `curl/flow.md` for a manual walkthrough of every
endpoint.
