# Proposal: Cross-DC Cache Synchronization

## 1. Problem Overview

The ToC (read-only) service experiences data inconsistency of up to 30 minutes due to reliance on cache TTLs. Updates made in DC A (SG) are replicated to DC B (US) via MySQL binlog, but no active cache invalidation mechanism exists for the local and Redis caches in DC B. The goal is to reduce this inconsistency to less than 5 minutes.

## 2. Proposed Solutions

Here are several potential solutions to address the cache synchronization issue:

### Solution 1: Pub/Sub Messaging System (e.g., RocketMQ or Kafka)

ToB services in DC A publish update events to a messaging system upon MySQL writes. ToC services in DC B subscribe to these events and invalidate corresponding entries in local caches and Redis.

- **Pros**: Real-time, scalable, decouples services, uses existing infrastructure.
- **Cons**: Added latency, potential message loss, complexity in failure handling.

### Solution 2: Binlog-Based Invalidation Trigger (Chosen Solution)

Extend the existing binlog replication from DC A to B by adding a consumer in DC B that parses binlog events for relevant table changes and triggers cache invalidation in ToC services.

- **Pros**: Leverages existing binlog sync, low overhead, reliable.
- **Cons**: Parsing complexity, potential delay in binlog replication, tight coupling to MySQL.

### Solution 3: Cross-DC Distributed Cache Replication

Configure Redis as a cross-DC replicated cluster where writes in DC A automatically propagate to DC B.

- **Pros**: Automatic synchronization, high availability.
- **Cons**: High complexity and cost, potential for eventual consistency issues.

### Solution 4: Periodic Polling/Sync Jobs

Implement scheduled jobs in DC B that query MySQL for recent changes and invalidate caches accordingly.

- **Pros**: Simple to implement, no new infrastructure.
- **Cons**: Not real-time, inefficient, increases DB load.

### Solution 5: Webhook/HTTP Callbacks

ToB services in DC A directly send HTTP requests to an endpoint in ToC services in DC B to invalidate specific cache keys.

- **Pros**: Straightforward for single updates, low latency if network is reliable.
- **Cons**: High cross-DC latency, SPOF, poor scalability.

### Solution 6: Stream Processing Pipeline (e.g., Flink or Spark)

Use a stream processor to consume binlog or message streams, process changes, and broadcast invalidation signals.

- **Pros**: Handles complex logic scalably, high reliability.
- **Cons**: Overkill for simple invalidation, adds significant complexity and cost.

## 3. Detailed Design for Solution 2: Binlog-Based Invalidation Trigger

This solution is divided into two main steps.

### Step 1: Reading Binlog to Detect MySQL Data Changes

#### Q1: In the ToB -> Cross-Region Binlog -> ToC setup, does the MySQL in DC B generate new binlogs after consuming and applying changes? Can we directly read binlogs in DC B to detect changes, and how to implement it logically?

**A**: By default, a MySQL slave does not generate new binlogs for replicated changes. This behavior is controlled by the `log-slave-updates` setting. If enabled on the DC B slave, it will log the applied changes. Assuming it is (or can be) enabled, you can and should read the binlog directly from the DC B MySQL instance to avoid cross-DC latency.

**Logical Implementation**:

1. Deploy a dedicated binlog consumer service in DC B.
2. This service connects to the DC B MySQL instance as a replication slave.
3. It streams binlog events, filtering for changes to relevant tables (e.g., `web_tracker`).
4. It parses these events to extract the changed data (e.g., primary keys).
5. The consumer must track and persist its binlog position to handle restarts gracefully.

#### Q2: What entity reads the binlog, from where, and how to parse it into logical data changes?

**A**:

- **Entity**: A dedicated, lightweight Golang microservice in DC B.
- **From**: The local MySQL instance in DC B.
- **Parsing**: Use a library like `github.com/siddontang/go-mysql`. The library decodes binary events into logical structures representing `INSERT`, `UPDATE`, and `DELETE` operations, including table names and row data.

#### Q3: How to ensure the order of reading MySQL data changes in this scenario?

**A**: Binlogs are inherently sequential. To maintain order:

1. Process events in a single thread within the consumer.
2. If scaling to multiple consumers, partition events by a consistent key (e.g., `tracker_id`) to ensure order within a partition.
3. Persist the last processed binlog position (file and offset, or GTID) to a durable store like Redis or a local file.

#### Follow-up on Q3: If the binlog consumer is part of the ToC service with hundreds of instances, how to ensure consumption order?

**A**: Integrating the consumer directly into each ToC instance is not recommended as it would lead to massive duplication and disorder. The correct approach is to have a small, separate cluster of consumer services. This cluster elects a single leader (using ZooKeeper, etcd, or a similar mechanism) that is responsible for processing the binlog. The leader then broadcasts invalidation messages to all ToC instances via a pub/sub system (like Redis Pub/Sub or Kafka). This decouples consumption from the ToC service and maintains global order.

#### Q4: What problems arise if there are a large number of data changes in a short time, and how to solve them?

**A**:

- **Problem**: Processing backlogs, resource overload on the consumer, and downstream pressure on Redis/ToC.
- **Solution**:
  - **Scale**: Horizontally scale the consumer service with sharding.
  - **Batching**: Group invalidation messages to reduce overhead.
  - **Backpressure**: Use bounded queues and rate limiting.
  - **Monitoring**: Alert on growing backlogs to trigger auto-scaling or manual intervention.

#### Q5: What other corner cases need to be considered?

**A**:

- **Replication Lag**: Monitor the replication lag between DC A and DC B.
- **Idempotency**: Ensure invalidation logic is idempotent to handle duplicate events.
- **Schema Changes**: The consumer must be able to handle DDL changes without crashing. This can be done by detecting DDL events and reloading the table schema.
- **Failures**: Handle MySQL connection failures, consumer restarts, and network partitions gracefully.

### Step 2: Invalidating Redis and Local Caches

#### Q1: How to invalidate distributed Redis and distributed Local Caches? Invalidate or update? Pros/Cons

**A**:

- **Invalidation**:
  - **Redis**: The consumer service issues `DEL` commands to the Redis cluster for the affected keys.
  - **Local Cache**: The consumer broadcasts invalidation messages (containing the keys) over a pub/sub channel. Each ToC instance subscribes to this channel and deletes the corresponding keys from its local cache.
- **Invalidate vs. Update**:
  - **Invalidate (Delete)**:
    - **Pros**: Simple, safe for complex data.
    - **Cons**: Can cause a "thundering herd" problem on the database.
  - **Update (Proactive Refresh)**:
    - **Pros**: Avoids cache misses, keeps hit rates high.
    - **Cons**: More complex, higher overhead, risk of propagating inconsistent data.
  - **Recommendation**: Start with **invalidation**. If the resulting database load is too high, consider more advanced strategies like stale-while-revalidate or switching to proactive updates.

#### Q2: If a large number of user requests hit the ToC service at the exact moment of cache invalidation, what happens and how to solve it?

**A**:

- **Problem**: A "thundering herd" of requests hits the database simultaneously, as they all miss the cache.
- **Solution**: Implement a **single-flight** mechanism (e.g., using Golang's `singleflight` package). This ensures that for a given key, only one request goes to the database; other concurrent requests for the same key wait for the result of that single request.

#### Q3: What is the order between Redis invalidation/update and Local Cache invalidation/update, and why design it that way?

**A**: The correct order is **Redis first, then Local Cache**.

- **Reasoning**: This follows the cache hierarchy. Redis acts as the shared, authoritative L2 cache. By invalidating it first, you ensure that any subsequent local cache miss will fetch fresh data from Redis (if available) or the database, preventing the local cache from being populated with stale data from Redis.

#### Q4: How to elegantly handle changes to the `web_tracker` table schema?

**A**:

- **DDL Detection**: The binlog consumer should detect DDL events (`ALTER TABLE`, etc.).
- **Schema Reloading**: Upon detecting a DDL event for a tracked table, the consumer should pause, reload the new schema from the database, and then resume processing using a parser that understands the new structure.
- **Versioning**: Maintain versions of your parsing logic to handle different schema versions gracefully during a transition.
- **Error Handling**: Log errors for unparseable events and alert developers, while attempting to skip non-critical events to avoid halting the entire pipeline.
