package cache_inconsistency

import (
	"SYS_DESIGN_PLAYGROUND/internal/registry"
	"SYS_DESIGN_PLAYGROUND/pkg/scenario"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
)

// Ensure CacheInconsistencyScenario implements the scenario.Scenario interface.
var _ scenario.Scenario = (*CacheInconsistencyScenario)(nil)

// init registers the scenario with the central registry.
func init() {
	registry.Register(&CacheInconsistencyScenario{})
}

const (
	productID   = 101
	productName = "Laptop"
)

var (
	db          *sql.DB
	redisClient *redis.Client
	ctx         = context.Background()
)

// product represents the data model for our product.
type product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// CacheInconsistencyScenario demonstrates the problem of write-through cache inconsistency.
type CacheInconsistencyScenario struct{}

func (s *CacheInconsistencyScenario) ID() string {
	return "cache_inconsistency"
}

func (s *CacheInconsistencyScenario) Name() string {
	return "DB/Cache Write Inconsistency"
}

func (s *CacheInconsistencyScenario) Category() string {
	return "Distributed Systems"
}

func (s *CacheInconsistencyScenario) ProblemDescription() string {
	return "When updating data, the write to the primary database succeeds, but the subsequent write to the cache fails. This leaves stale data in the cache. Future reads will hit the cache and serve incorrect, outdated information, leading to data inconsistency between the DB and cache."
}

func (s *CacheInconsistencyScenario) SolutionDescription() string {
	return "A common and robust solution is the 'Cache-Aside' pattern combined with a 'Write-Through' strategy where the cache key is deleted instead of updated. On write, you update the database and then invalidate the cache by deleting the corresponding key. If the deletion fails, the stale cache entry will eventually be removed when its Time-To-Live (TTL) expires. On read, if the data is not in the cache (a 'cache miss'), you fetch it from the database, store it in the cache with a TTL, and then return it."
}

func (s *CacheInconsistencyScenario) DeepDiveLink() string {
	return "https://redis.io/docs/latest/develop/get-started/patterns/cache-aside/"
}

func (s *CacheInconsistencyScenario) Actions() []scenario.Action {
	return []scenario.Action{
		{ID: "update_naive", Name: "Update Price (Problematic)", Description: "Updates the DB, then attempts to update the cache, but the cache update will fail."},
		{ID: "update_with_fix", Name: "Update Price (Solution)", Description: "Updates the DB, then invalidates the cache by deleting the key."},
		{ID: "reset", Name: "Reset State", Description: "Resets the product price and clears the cache."},
	}
}

func (s *CacheInconsistencyScenario) DashboardComponents() []scenario.DashboardComponent {
	return []scenario.DashboardComponent{
		{ID: "mysql_record", Name: "MySQL Product Record", Type: "key_value"},
		{ID: "redis_key", Name: "Redis Cache Key", Type: "key_value"},
		{ID: "logs", Name: "Live Logs", Type: "log_stream"},
	}
}

// Initialize connects to the database and Redis, and sets up the initial state.
func (s *CacheInconsistencyScenario) Initialize() error {
	var err error
	// Connect to MySQL
	db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/playground")
	if err != nil {
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql: %w", err)
	}

	// Connect to Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if _, err = redisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Setup schema and initial data
	return s.resetState()
}

func (s *CacheInconsistencyScenario) ExecuteAction(actionID string, params map[string]interface{}) (interface{}, error) {
	switch actionID {
	case "update_naive":
		return s.updateNaive()
	case "update_with_fix":
		return s.updateWithFix()
	case "reset":
		return "State reset", s.resetState()
	default:
		return nil, fmt.Errorf("unknown action: %s", actionID)
	}
}

func (s *CacheInconsistencyScenario) FetchState() (map[string]interface{}, error) {
	// Fetch from DB
	var p product
	err := db.QueryRow("SELECT id, name, price FROM products WHERE id = ?", productID).Scan(&p.ID, &p.Name, &p.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from db: %w", err)
	}

	// Fetch from Redis
	cacheKey := fmt.Sprintf("product:%d", productID)
	val, err := redisClient.Get(ctx, cacheKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to fetch from redis: %w", err)
	}
	if err == redis.Nil {
		val = "null (cache miss)"
	}

	return map[string]interface{}{
		"mysql_record": p,
		"redis_key":    val,
	}, nil
}

// updateNaive demonstrates the problem: DB write succeeds, cache write fails.
func (s *CacheInconsistencyScenario) updateNaive() (string, error) {
	log.Println("Executing naive update...")
	newPrice := 99.99
	// 1. Update database
	_, err := db.Exec("UPDATE products SET price = ? WHERE id = ?", newPrice, productID)
	if err != nil {
		log.Printf("ERROR: Failed to update database: %v", err)
		return "Failed to update database", err
	}
	log.Printf("SUCCESS: Database updated. Price set to %.2f", newPrice)

	// 2. Simulate a failure to update cache
	log.Println("ATTEMPT: Updating cache...")
	log.Println("ERROR: Cache update failed!")
	return "DB updated, but cache update failed, causing inconsistency.", errors.New("simulated cache update failure")
}

// updateWithFix demonstrates the solution: update DB, then invalidate cache.
func (s *CacheInconsistencyScenario) updateWithFix() (string, error) {
	log.Println("Executing solution update...")
	newPrice := 129.99
	// 1. Update database
	_, err := db.Exec("UPDATE products SET price = ? WHERE id = ?", newPrice, productID)
	if err != nil {
		log.Printf("ERROR: Failed to update database: %v", err)
		return "Failed to update database", err
	}
	log.Printf("SUCCESS: Database updated. Price set to %.2f", newPrice)

	// 2. Invalidate cache by deleting the key
	log.Println("ATTEMPT: Invalidating cache by deleting key...")
	cacheKey := fmt.Sprintf("product:%d", productID)
	if err := redisClient.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("ERROR: Failed to invalidate cache: %v", err)
		// Even if this fails, the TTL will eventually save us.
		return "DB updated, but failed to invalidate cache.", err
	}
	log.Println("SUCCESS: Cache invalidated.")
	return "DB updated and cache invalidated successfully.", nil
}

// resetState sets up the initial database table and data.
func (s *CacheInconsistencyScenario) resetState() error {
	// Create table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INT PRIMARY KEY,
			name VARCHAR(255),
			price DECIMAL(10, 2)
		)
	`)
	if err != nil {
		return err
	}

	// Reset or insert data
	initialPrice := 79.99
	_, err = db.Exec(`
		INSERT INTO products (id, name, price) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE name = ?, price = ?
	`, productID, productName, initialPrice, productName, initialPrice)
	if err != nil {
		return err
	}

	// Prime the cache
	p := product{ID: productID, Name: productName, Price: initialPrice}
	pJSON, _ := json.Marshal(p)
	cacheKey := fmt.Sprintf("product:%d", productID)
	return redisClient.Set(ctx, cacheKey, pJSON, 10*time.Minute).Err()
}