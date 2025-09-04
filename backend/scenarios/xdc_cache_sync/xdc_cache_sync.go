package xdccachesync

import (
	"SYS_DESIGN_PLAYGROUND/internal/registry"
	"SYS_DESIGN_PLAYGROUND/pkg/repo/model/model"
	"SYS_DESIGN_PLAYGROUND/pkg/scenario"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
)

var _ scenario.Scenario = (*XDCCacheSyncScenario)(nil)

func init() {
	registry.Register(&XDCCacheSyncScenario{})
}

type XDCCacheSyncScenario struct {
	db          *sql.DB
	redisClient *redis.Client
	localCache  *LocalCache
	cacheMgr    *CacheManager

	binlogListener    *BinlogListener
	rocketmqProcessor *CDCEventProcessor
	rocketmqManager   *RocketMQManager

	mu            sync.RWMutex
	logs          []string
	testProductID int64
	ctx           context.Context
}

func (s *XDCCacheSyncScenario) ID() string {
	return "xdc_cache_sync"
}

func (s *XDCCacheSyncScenario) Name() string {
	return "Cross-DC Cache Synchronization"
}

func (s *XDCCacheSyncScenario) Category() string {
	return "Distributed Systems"
}

func (s *XDCCacheSyncScenario) ProblemDescription() string {
	return "In a multi-DC setup, writes happen in DC A but reads happen in DC B. When data is updated in DC A, the caches (Redis + local) in DC B become stale for up to 30 minutes due to TTL expiration. This causes inconsistent user experience as ToC service in DC B serves outdated data."
}

func (s *XDCCacheSyncScenario) SolutionDescription() string {
	return "Implement binlog-based cache invalidation using CDC (Canal) to capture changes from MySQL replica in DC B, publish to message queue (Kafka), and invalidate both Redis and local caches proactively. This reduces data inconsistency from 30 minutes to under 5 minutes while maintaining high availability."
}

func (s *XDCCacheSyncScenario) DeepDiveLink() string {
	return "https://github.com/alibaba/canal"
}

func (s *XDCCacheSyncScenario) Actions() []scenario.Action {
	return []scenario.Action{
		{ID: "initialize", Name: "Initialize System", Description: "启动 Mysql LocalCache Redis BinlogListener 和 CacheInvalidationEventProcessor"},
		{ID: "read_first", Name: "Read First", Description: "创建一条web_product测试数据(如果不存在的话) 读取这条数据 同时确保数据能从 Mysql 填充到 Redis和 LocalCache"},
		{ID: "update_record", Name: "Update Record", Description: "更新测试数据的 extra 字段 触发mysql 的 Binlog, BinlogListener 会接收这条 Binlog 并转换成 CDCEvent"},
		{ID: "read_second", Name: "Read Second", Description: "再次读取测试记录 预期会直接读取 Mysql得到最新的结果(同时也会将新结果填充到 Redis 和 Local Cache)"},
	}
}

func (s *XDCCacheSyncScenario) DashboardComponents() []scenario.DashboardComponent {
	return []scenario.DashboardComponent{
		{ID: "mysql_record", Name: "MySQL Product Record", Type: "key_value"},
		{ID: "redis_cache", Name: "Redis Cache", Type: "key_value"},
		{ID: "local_cache", Name: "Local Cache", Type: "key_value"},
		{ID: "cache_stats", Name: "Cache Statistics", Type: "key_value"},
		{ID: "logs", Name: "Live Logs", Type: "log_stream"},
	}
}

func (s *XDCCacheSyncScenario) Initialize() error {
	s.ctx = context.Background()
	s.testProductID = 10001
	s.logs = make([]string, 0)

	var err error
	// Connect to MySQL
	s.db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/playground")
	if err != nil {
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}
	if err = s.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql: %w", err)
	}
	s.addLog("Connected to MySQL successfully")

	// Connect to Redis
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if _, err = s.redisClient.Ping(s.ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	s.addLog("Connected to Redis successfully")

	// Initialize LocalCache
	s.localCache = NewLocalCache()
	s.addLog("LocalCache initialized")

	// Initialize CacheManager
	s.cacheMgr = NewCacheManager(s.redisClient, s.localCache)
	s.addLog("CacheManager initialized")

	return nil
}

func (s *XDCCacheSyncScenario) ExecuteAction(actionID string, params map[string]interface{}) (interface{}, error) {
	switch actionID {
	case "initialize":
		return s.initializeSystem()
	case "read_first":
		return s.readFirst()
	case "update_record":
		return s.updateRecord()
	case "read_second":
		return s.readSecond()
	default:
		return nil, fmt.Errorf("unknown action: %s", actionID)
	}
}

func (s *XDCCacheSyncScenario) FetchState() (map[string]interface{}, error) {
	return nil, nil
}

func (s *XDCCacheSyncScenario) addLog(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timestamp := time.Now().Format("15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)
	s.logs = append(s.logs, logEntry)
	log.Println(logEntry)
}

func (s *XDCCacheSyncScenario) initializeSystem() (string, error) {
	s.addLog("Starting BinlogListener...")

	// Create canal config
	cfg := CreateDefaultCanalConfig()

	// Create and start binlog listener
	var err error
	s.binlogListener, err = NewBinlogListener(cfg)
	if err != nil {
		s.addLog(fmt.Sprintf("Failed to create BinlogListener: %v", err))
		return "Failed to create BinlogListener", err
	}

	// Add table filter for web_product
	s.binlogListener.AddTableFilter("playground", "web_product")

	// Start binlog listener
	if err := s.binlogListener.Start(); err != nil {
		s.addLog(fmt.Sprintf("Failed to start BinlogListener: %v", err))
		return "Failed to start BinlogListener", err
	}
	s.addLog("BinlogListener started successfully")

	// Start CDC Event Processor
	s.rocketmqProcessor = &CDCEventProcessor{
		eventChan:    s.binlogListener.GetEventChannel(),
		cacheManager: s.cacheMgr,
		stopChan:     make(chan struct{}),
		running:      true,
	}

	go s.startCDCEventProcessor()
	s.addLog("CacheInvalidationEventProcessor started")

	// Initialize RocketMQ Manager
	var err2 error
	s.rocketmqManager, err2 = NewRocketMQManager(
		[]string{"rocketmq-nameserver:9876"}, // RocketMQ nameserver address
		"cache_invalidation_topic",
		"cache_invalidation_group",
	)
	if err2 != nil {
		s.addLog(fmt.Sprintf("Failed to create RocketMQ manager: %v", err2))
		return "Failed to create RocketMQ manager", err2
	}

	// Start RocketMQ producer
	if err2 := s.rocketmqManager.StartProducer(); err2 != nil {
		s.addLog(fmt.Sprintf("Failed to start RocketMQ producer: %v", err2))
		return "Failed to start RocketMQ producer", err2
	}
	s.addLog("RocketMQ producer started")

	// Start RocketMQ consumer
	if err2 := s.rocketmqManager.StartConsuming(s.handleInvalidationMessage); err2 != nil {
		s.addLog(fmt.Sprintf("Failed to start RocketMQ consumer: %v", err2))
		return "Failed to start RocketMQ consumer", err2
	}
	s.addLog("RocketMQ consumer started")

	return "System initialized successfully", nil
}

// readFirst creates test data and reads it, populating both Redis and LocalCache
func (s *XDCCacheSyncScenario) readFirst() (string, error) {
	s.addLog("Creating/Reading test web_product data...")

	// Create test product if not exists
	testProduct := &model.WebProduct{
		ID:      s.testProductID,
		Code:    "TEST_PRODUCT_001",
		Name:    "Test Product for XDC Cache Sync",
		Mode:    1,
		Extra:   `{"description": "Initial test data", "version": 1}`,
		Version: 1,
	}

	// Insert or update test data
	_, err := s.db.Exec(`
		INSERT INTO web_product (id, code, name, mode, extra, version)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		name = VALUES(name), mode = VALUES(mode), extra = VALUES(extra), version = VALUES(version)
	`, testProduct.ID, testProduct.Code, testProduct.Name, testProduct.Mode, testProduct.Extra, testProduct.Version)

	if err != nil {
		s.addLog(fmt.Sprintf("Failed to create test data: %v", err))
		return "Failed to create test data", err
	}
	s.addLog("Test data created/updated in MySQL")

	// Read from MySQL and populate caches
	product, err := s.readProductWithCaching(s.testProductID)
	if err != nil {
		s.addLog(fmt.Sprintf("Failed to read product: %v", err))
		return "Failed to read product", err
	}

	s.addLog(fmt.Sprintf("Product read successfully: %+v", product))
	s.addLog("Data populated to both Redis and LocalCache")

	return fmt.Sprintf("Product read: %+v", product), nil
}

// updateRecord updates the extra field to trigger binlog
func (s *XDCCacheSyncScenario) updateRecord() (string, error) {
	s.addLog("Updating test product extra field...")

	newExtra := fmt.Sprintf(`{"description": "Updated test data", "version": 2, "timestamp": "%s"}`, time.Now().Format(time.RFC3339))

	_, err := s.db.Exec(`UPDATE web_product SET extra = ?, version = version + 1 WHERE id = ?`, newExtra, s.testProductID)
	if err != nil {
		s.addLog(fmt.Sprintf("Failed to update product: %v", err))
		return "Failed to update product", err
	}

	s.addLog("Product updated in MySQL, binlog event should be triggered")
	s.addLog("Waiting for CDC event processing...")

	// Give some time for binlog processing
	time.Sleep(2 * time.Second)

	return "Product updated, binlog event triggered", nil
}

// readSecond reads the updated data, should get latest from MySQL
func (s *XDCCacheSyncScenario) readSecond() (string, error) {
	s.addLog("Reading product after update...")

	product, err := s.readProductWithCaching(s.testProductID)
	if err != nil {
		s.addLog(fmt.Sprintf("Failed to read updated product: %v", err))
		return "Failed to read updated product", err
	}

	s.addLog(fmt.Sprintf("Updated product read: %+v", product))
	s.addLog("Fresh data retrieved from MySQL and cached")

	return fmt.Sprintf("Updated product: %+v", product), nil
}

// readProductWithCaching implements cache-aside pattern
func (s *XDCCacheSyncScenario) readProductWithCaching(productID int64) (*model.WebProduct, error) {
	cacheKey := fmt.Sprintf("web_product:%d", productID)

	// 1. Try LocalCache first
	if val, found := s.localCache.Get(cacheKey); found {
		s.cacheMgr.IncrementLocalHit()
		s.addLog("Cache HIT: LocalCache")
		if product, ok := val.(*model.WebProduct); ok {
			return product, nil
		}
	}
	s.cacheMgr.IncrementLocalMiss()

	// 2. Try Redis
	val, err := s.redisClient.Get(s.ctx, cacheKey).Result()
	if err == nil {
		s.cacheMgr.IncrementRedisHit()
		s.addLog("Cache HIT: Redis")

		var product model.WebProduct
		if err := json.Unmarshal([]byte(val), &product); err == nil {
			s.localCache.Set(cacheKey, &product)
			return &product, nil
		}
	}
	s.cacheMgr.IncrementRedisMiss()

	// 3. Query MySQL
	s.cacheMgr.IncrementDBQuery()
	s.addLog("Cache MISS: Querying MySQL")

	var product model.WebProduct
	var createdAt, updatedAt, deletedAt sql.NullTime

	err = s.db.QueryRow(`
		SELECT id, code, name, mode, extra, version, created_at, updated_at, deleted_at
		FROM web_product WHERE id = ?
	`, productID).Scan(
		&product.ID, &product.Code, &product.Name, &product.Mode, &product.Extra,
		&product.Version, &createdAt, &updatedAt, &deletedAt,
	)

	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		product.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		product.UpdatedAt = updatedAt.Time
	}

	// 4. Cache the result
	productJSON, _ := json.Marshal(product)
	s.redisClient.Set(s.ctx, cacheKey, productJSON, 10*time.Minute)
	s.localCache.Set(cacheKey, &product)
	s.addLog("Data cached to both Redis and LocalCache")

	return &product, nil
}

// startCDCEventProcessor processes CDC events from binlog
func (s *XDCCacheSyncScenario) startCDCEventProcessor() {
	s.addLog("CDC Event Processor started")

	for {
		select {
		case event := <-s.rocketmqProcessor.eventChan:
			s.processCDCEvent(event)
		case <-s.rocketmqProcessor.stopChan:
			s.addLog("CDC Event Processor stopped")
			return
		}
	}
}

// processCDCEvent handles individual CDC events
func (s *XDCCacheSyncScenario) processCDCEvent(event *CDCEvent) {
	s.addLog(fmt.Sprintf("Processing CDC Event: %s %s.%s", event.Operation, event.Schema, event.Table))

	if event.Table != "web_product" {
		return
	}

	// Extract product ID from primary key
	productID, ok := event.PrimaryKey["id"]
	if !ok {
		s.addLog("Failed to extract product ID from CDC event")
		return
	}

	cacheKey := fmt.Sprintf("web_product:%v", productID)

	// 1. Delete from Redis
	if err := s.redisClient.Del(s.ctx, cacheKey).Err(); err != nil {
		s.addLog(fmt.Sprintf("Failed to delete Redis key %s: %v", cacheKey, err))
	} else {
		s.addLog(fmt.Sprintf("Deleted Redis key: %s", cacheKey))
	}

	// Send broadcast message via RocketMQ
	invalidationMsg := &InvalidationMessage{
		Timestamp: time.Now(),
		Reason:    "cdc-invalidated",
		Table:     event.Table,
		Keys:      []string{cacheKey},
		Version:   "1.0",
		TraceID:   fmt.Sprintf("cdc-%d", time.Now().UnixNano()),
	}

	if err := s.rocketmqManager.SendInvalidationMessage(invalidationMsg); err != nil {
		s.addLog(fmt.Sprintf("Failed to send invalidation message: %v", err))
	} else {
		s.addLog(fmt.Sprintf("Sent invalidation message via RocketMQ: %s", cacheKey))
	}
}

// handleInvalidationMessage processes cache invalidation messages from RocketMQ
func (s *XDCCacheSyncScenario) handleInvalidationMessage(msg *InvalidationMessage) error {
	s.addLog(fmt.Sprintf("Received invalidation message: table=%s, keys=%v, reason=%s",
		msg.Table, msg.Keys, msg.Reason))

	// Delete from local cache
	for _, key := range msg.Keys {
		s.localCache.Delete(key)
		s.addLog(fmt.Sprintf("Deleted from LocalCache: %s", key))
	}

	return nil
}
