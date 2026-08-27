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

## Distributed lock: Redis vs. a DB-level lock

- **What this project ships**: a single `SET NX PX` (via redsync's
  `TryLockContext`, one attempt, no retry loop) against one Redis node.
  Correct mutual exclusion in normal operation, and safe against an
  orphaned lock because the TTL reclaims it - but this is not full
  Redlock. Real Redlock requires acquiring the lock on a majority of N
  (typically 5) independently run Redis masters within a tight time
  budget, so the lock survives any single node dying or partitioning.
  This project's one-node setup is a single point of failure for the
  lock, the same way it already is for any cache - worth calling out
  explicitly rather than glossing over.
- **Known Redlock caveats, unresolved by design**: a client that stalls
  past the lock's TTL (a GC pause, a suspended VM) can wake up still
  believing it holds the lock after another client has already acquired
  it - Redlock alone gives no fencing token a downstream resource can
  check. A replica promoted to master after asynchronously replicating a
  lock write can also let two clients hold the "same" lock at once.
- **When to reach for this anyway**: best-effort de-duplication where an
  occasional double-run is tolerable - skip a duplicate cron tick,
  coalesce a cache rebuild. That is what `/jobs/run` demonstrates here.
- **When to reach for a DB-level lock instead**: any invariant where a
  duplicate execution is a correctness bug (double-spend, double
  allocation of a unique resource). A database already gives an atomic,
  durable, replicated decision point (`SELECT ... FOR UPDATE`, a unique
  constraint, a Postgres advisory lock) with fencing built into the
  transaction itself - reach for that before layering token-based
  fencing on top of Redis.

## Rate limiter design

Sliding-window log: one Redis sorted set per client, trimmed and checked
inside a single Lua script (`ZREMRANGEBYSCORE` to evict anything older
than the window, `ZCARD` to count, `ZADD` + `PEXPIRE` to record the
request), so the whole decision is atomic. This avoids the double-limit
burst a fixed-window `INCR`+`EXPIRE` allows at window boundaries, and
gives an exact window rather than a token bucket's floating-point refill
math. `X-RateLimit-Remaining` and `X-RateLimit-Reset` are set from the
same script result - `Reset` comes from the oldest surviving request's
timestamp, not `now + window`, since capacity frees up when that request
ages out.

## Tests

```
go test ./...
go generate ./...   # regenerate repository/service mocks
```

Integration tests (`*_integration_test.go` at the project root) need a
real Redis on `localhost:6379` and skip automatically if it is
unreachable. See `curl/flow.md` for a manual walkthrough of every
endpoint.
