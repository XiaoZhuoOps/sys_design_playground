package xdccachesync

import (
	"time"

	"github.com/apache/rocketmq-client-go/v2"
)

type CDCEvent struct {
	Timestamp  time.Time              `json:"timestamp"`
	Schema     string                 `json:"schema"`
	Table      string                 `json:"table"`
	Operation  string                 `json:"operation"` // INSERT, UPDATE, DELETE
	PrimaryKey map[string]interface{} `json:"primary_key"`
	Before     map[string]interface{} `json:"before"`
	After      map[string]interface{} `json:"after"`
	GTID       string                 `json:"gtid,omitempty"`
	TxID       string                 `json:"tx_id,omitempty"`
}

type InvalidationMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason"` // e.g., "cdc-invalidated"
	Table     string    `json:"table"`
	Keys      []string  `json:"keys"`
	Version   string    `json:"version"`
	TraceID   string    `json:"trace_id,omitempty"`
}

type CacheStats struct {
	LocalHits   int64 `json:"local_hits"`
	LocalMisses int64 `json:"local_misses"`
	RedisHits   int64 `json:"redis_hits"`
	RedisMisses int64 `json:"redis_misses"`
	DBQueries   int64 `json:"db_queries"`
}

type CDCEventProcessor struct {
	eventChan    <-chan *CDCEvent
	cacheManager *CacheManager
	stopChan     chan struct{}
	running      bool
}

type RocketMQManager struct {
	producer rocketmq.Producer
	consumer rocketmq.PushConsumer
	topic    string
	group    string
}

type MessageProducer interface {
	SendInvalidationMessage(msg *InvalidationMessage) error
}

type MessageConsumer interface {
	StartConsuming(handler func(*InvalidationMessage) error) error
	Stop() error
}
