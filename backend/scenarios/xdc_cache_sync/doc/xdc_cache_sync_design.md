# Cross-DC Cache Synchronization Design (Canal-driven Invalidation)

1. Summary

- Problem: Up to 30 minutes of stale data in DC B ToC due to cache TTLs; updates originate in DC A and replicate to DC B via MySQL binlog, but no active cache invalidation for Redis and local caches in DC B.
- Goal: Reduce inconsistency to under 5 minutes (P99) without introducing SPOF, and keep added read-path latency under 100 ms.
- Approach: Use binlog-based cache invalidation in DC B driven by Canal (HA), with a durable message bus (Kafka/RocketMQ) and a horizontally scalable invalidation service. Invalidate Redis first, then local caches; rely on idempotency and at-least-once semantics.

2. Scope and Non-Goals

- In scope: ToB writes in DC A; A→B replication; DC B CDC (Canal); durable fan-out; Redis and local cache invalidation; ToC read-path adaptations (singleflight/stale-while-revalidate).
- Non-goals: Write-side changes to ToB; Redis cross-DC replication; business-layer schema redesign.

3. Architecture Overview

- DC A: MySQL primary with row-based binlog enabled, GTID enabled.
- DC B: MySQL replica that applies A’s changes; log_slave_updates enabled so applied changes are also written to B’s local binlogs.
- DC B: Canal cluster (multiple canal-server instances) with ZooKeeper for leader election and metadata (binlog positions and schema history), pulling binlogs from DC B MySQL.
- DC B: MQ (Kafka/RocketMQ) as persistent, replayable transport; topics partitioned by table/key to preserve per-key order and allow horizontal scaling.
- DC B: Invalidation workers (stateless) consume events, map them to cache keys, batch/pipeline delete in Redis Cluster, and broadcast local cache invalidation to all ToC instances.
- DC B: ToC service instances subscribe to a local invalidation channel to evict in-memory caches; read path uses singleflight (and optionally stale-while-revalidate).

4. Data Flow (End-to-End)

- Step 1 (Write): ToB updates MySQL in DC A.
- Step 2 (Replication): Binlog-based replication propagates to DC B (typical lag ~200 ms; monitor and budget up to seconds).
- Step 3 (CDC): Canal leader for the destination in DC B connects to B’s MySQL, reads binlogs (GTID or file+offset), parses row events, and converts to normalized change messages (table, op, PK, columns, ts, txid/gtid).
- Step 4 (Fan-out): Canal publishes messages to MQ with partitioning by key (e.g., hash(schema.table:pk)).
- Step 5 (Invalidation): Workers consume, coalesce duplicate keys within a short time window, delete keys in Redis (pipeline/UNLINK), then publish a local invalidation message (keys[]) that all ToC instances subscribe to.
- Step 6 (ToC Read): ToC instances evict local caches. Subsequent requests miss local cache; if Redis miss, they hit DB, guarded by singleflight (and optionally stale-while-revalidate).

5. Ordering and Consistency Model

- Binlog guarantees transactional order; Canal enforces a single consumer (leader) per destination to preserve stream order.
- MQ preserves per-partition order; using a partition key of schema.table:pk ensures same-key ordering, which is sufficient for cache invalidation.
- At-least-once semantics: failures or failover can replay the last acknowledged chunk; invalidation operations are idempotent.
- Global total order is not guaranteed (nor required); same-key order is guaranteed.
- Fallback consistency: TTLs remain as a safety net; when inflight failures occur, maximum staleness bounded by TTL.

6. High Availability and Failover

- MySQL: GTID enabled; DC B replica configured with log_slave_updates and row-based binlog.
- Canal: Multiple canal-server instances with ZooKeeper. Only one leader per destination reads; others are hot standby. Meta (binlog position) stored in ZK (or mixed mode) for quick takeover with at-least-once replay.
- MQ: Kafka with acks=all, replication factor ≥3, min.insync.replicas tuned to tolerate broker loss.
- Workers: Stateless; scale by partitions; rolling upgrades with consumer group rebalancing.
- ToC invalidation channel: Prefer Kafka to avoid message loss and allow replay; Redis Pub/Sub acceptable for MVP but not for sustained backlogs.

7. Component Design and Key Configurations

- MySQL DC A and B
  - gtid_mode=ON; enforce_gtid_consistency=ON
  - binlog_format=ROW; binlog_row_image=MINIMAL (ensure PKs present)
  - On DC B: log_bin=ON; log_slave_updates=ON
- Canal Cluster (DC B)
  - HA: ZooKeeper ensemble for leader election and meta
  - Destination granularity: per database or per business domain; split if throughput demands
  - TSDB (schema history) enabled to handle DDL
  - MQ sink: Kafka or RocketMQ; flatMessage=true for generic JSON payloads
  - Partitioning: partitionHash = schema.table:id to keep same-key order
- MQ (Kafka as reference)
  - Topic per table or domain; partitions sized for target throughput
  - acks=all; linger.ms to batch; compression (lz4/snappy) to reduce bandwidth
  - Retention sufficient for recovery/replay (hours–days)
- Invalidation Workers
  - Consume in consumer groups; concurrency per partition
  - Coalesce window 10–50 ms; bounded queues; backpressure and rate limiting
  - Redis pipeline with batch size tuned; prefer DEL for correctness; UNLINK to offload freeing
  - Publish local invalidation after Redis deletes to avoid rehyrating from stale Redis
- Redis Cluster
  - Key design: toc:{table}:v{N}:{pk}
  - Expiration: keep TTLs (e.g., 30 min) for safety; consider SWR for hot keys
  - Commands: DEL/UNLINK via pipelining/Lua to reduce round trips
- ToC Service
  - Local cache with subscription to invalidation topic/channel
  - Singleflight to avoid thundering herd; optional stale-while-revalidate for hot keys
  - Retry/backoff for subscription reconnects

8. Performance and Capacity (Guidance)

- Throughput
  - Canal parsing is rarely the bottleneck; worker and Redis operations usually dominate
  - Single worker can issue tens of thousands of Redis DEL/s with pipelining on adequate hardware
- Latency Budget (typical)
  - A→B replication: ~0.2–1 s (p50–p95)
  - Canal parse + MQ publish: <0.2 s
  - Worker consume + Redis delete + local broadcast: <0.2–0.5 s
  - P99 under burst may reach seconds; P99.9 still bounded by minutes with backlogs; TTL bounds worst-case
- Scaling
  - Increase Kafka partitions and worker instances to scale
  - If large-scale table-wide updates occur, consider namespace versioning (epoch) to avoid deleting millions of keys individually

9. Failure Modes and Mitigations

- Replication lag spikes: Apply backpressure; expose lag; allow TTL to cap staleness
- Canal leader crash: ZK failover; at-least-once replay; idempotent invalidation
- MQ backlog: Scale partitions/consumers; coalesce more aggressively; rate-limit invalidation to protect Redis
- Redis overload: Reduce batch size or pace; use UNLINK; temporarily skip local invalidation and rely on TTL (as a controlled degradation)
- DDL: Canal TSDB reloads; if parser errors occur, pause destination, refresh schema, resume; while paused, TTL provides safety
- Message duplication: Invalidation is idempotent; duplicates are safe
- Hot keys: Use singleflight and SWR; consider per-key rate limits

10. Security and Compliance

- TLS for MySQL, MQ, and Redis (where supported)
- Least-privilege credentials for Canal, workers, and ToC subscribers
- Topic- and channel-level ACLs; audit logs retained as per policy

11. Tradeoffs and Alternatives

- Canal vs Debezium: Debezium offers stronger schema history and Connect-managed HA; Canal is simple, widely used, and integrates with RocketMQ/Kafka
- Kafka vs Redis Pub/Sub: Kafka is durable and supports replay/backpressure; Redis Pub/Sub is simple but lossy and not suitable for backlogs
- Invalidate vs proactive refresh: Invalidate is simple and safe; proactive refresh reduces misses but risks propagating partial states and adds complexity
- Fine-grained deletes vs namespace epoch: Fine-grained is precise but costly for mass updates; epoch is coarse but efficient for bulk changes

12. Open Questions (to finalize)

- Final choice of MQ (Kafka vs RocketMQ) and partitioning plan per table/domain
- Whether to enable stale-while-revalidate for hot keys in ToC now or later
- Namespace versioning (epoch) policy for massive updates (on/off, thresholds)
