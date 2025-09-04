package xdccachesync

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/go-mysql-org/go-mysql/canal"
	_ "github.com/go-sql-driver/mysql"
)

func TestBinlogListenerWithRealDatabase(t *testing.T) {
	fmt.Println("🚀 开始测试 BinlogListener...")
	
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/playground?parseTime=true")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✅ 数据库连接成功")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_product (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) DEFAULT 0.00,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer db.Exec("DROP TABLE IF EXISTS test_product")
	fmt.Println("✅ 测试表创建成功")

	cfg := &canal.Config{
		Addr:     "localhost:3306",
		User:     "root",
		Password: "root",
		Charset:  "utf8mb4",
		Flavor:   "mysql",
		ServerID: 1001,
	}

	listener, err := NewBinlogListener(cfg)
	if err != nil {
		t.Fatalf("Failed to create binlog listener: %v", err)
	}
	defer listener.Stop()

	listener.AddTableFilter("playground", "test_product")

	if err := listener.Start(); err != nil {
		t.Fatalf("Failed to start binlog listener: %v", err)
	}
	fmt.Println("✅ BinlogListener 启动成功")

	time.Sleep(1 * time.Second)

	eventChan := listener.GetEventChannel()

	fmt.Println("\n📋 开始执行测试用例...")
	fmt.Println("=========================================")
	
	testInsert(t, db, eventChan)
	testUpdate(t, db, eventChan)
	testDelete(t, db, eventChan)
	
	fmt.Println("=========================================")
	fmt.Println("🎉 所有测试用例执行完成!")
}

func testInsert(t *testing.T, db *sql.DB, eventChan <-chan *CDCEvent) {
	fmt.Println("\n📝 测试 INSERT 操作...")
	fmt.Printf("执行 SQL: INSERT INTO test_product (name, price) VALUES ('Test Product', 99.99)\n")
	
	_, err := db.Exec("INSERT INTO test_product (name, price) VALUES (?, ?)", "Test Product", 99.99)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	select {
	case event := <-eventChan:
		fmt.Printf("📢 收到 CDC 事件:\n")
		fmt.Printf("  操作类型: %s\n", event.Operation)
		fmt.Printf("  表名: %s.%s\n", event.Schema, event.Table)
		fmt.Printf("  时间戳: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
		if event.After != nil {
			fmt.Printf("  新数据: %+v\n", event.After)
		}
		if event.PrimaryKey != nil {
			fmt.Printf("  主键: %+v\n", event.PrimaryKey)
		}
		
		if event.Operation != "INSERT" {
			t.Errorf("Expected INSERT operation, got %s", event.Operation)
		}
		if event.Table != "test_product" {
			t.Errorf("Expected test_product table, got %s", event.Table)
		}
		if event.After["name"] != "Test Product" {
			t.Errorf("Expected product name 'Test Product', got %v", event.After["name"])
		}
		fmt.Println("✅ INSERT 测试通过")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for INSERT event")
	}
}

func testUpdate(t *testing.T, db *sql.DB, eventChan <-chan *CDCEvent) {
	fmt.Println("\n🔄 测试 UPDATE 操作...")
	fmt.Printf("执行 SQL: UPDATE test_product SET name = 'Updated Product', price = 149.99 WHERE name = 'Test Product'\n")
	
	_, err := db.Exec("UPDATE test_product SET name = ?, price = ? WHERE name = ?", "Updated Product", 149.99, "Test Product")
	if err != nil {
		t.Fatalf("Failed to update test data: %v", err)
	}

	select {
	case event := <-eventChan:
		fmt.Printf("📢 收到 CDC 事件:\n")
		fmt.Printf("  操作类型: %s\n", event.Operation)
		fmt.Printf("  表名: %s.%s\n", event.Schema, event.Table)
		fmt.Printf("  时间戳: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
		if event.Before != nil {
			fmt.Printf("  修改前数据: %+v\n", event.Before)
		}
		if event.After != nil {
			fmt.Printf("  修改后数据: %+v\n", event.After)
		}
		if event.PrimaryKey != nil {
			fmt.Printf("  主键: %+v\n", event.PrimaryKey)
		}
		
		if event.Operation != "UPDATE" {
			t.Errorf("Expected UPDATE operation, got %s", event.Operation)
		}
		if event.Table != "test_product" {
			t.Errorf("Expected test_product table, got %s", event.Table)
		}
		fmt.Println("✅ UPDATE 测试通过")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for UPDATE event")
	}
}

func testDelete(t *testing.T, db *sql.DB, eventChan <-chan *CDCEvent) {
	fmt.Println("\n🗑️ 测试 DELETE 操作...")
	fmt.Printf("执行 SQL: DELETE FROM test_product WHERE name = 'Updated Product'\n")
	
	_, err := db.Exec("DELETE FROM test_product WHERE name = ?", "Updated Product")
	if err != nil {
		t.Fatalf("Failed to delete test data: %v", err)
	}

	select {
	case event := <-eventChan:
		fmt.Printf("📢 收到 CDC 事件:\n")
		fmt.Printf("  操作类型: %s\n", event.Operation)
		fmt.Printf("  表名: %s.%s\n", event.Schema, event.Table)
		fmt.Printf("  时间戳: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
		if event.Before != nil {
			fmt.Printf("  删除前数据: %+v\n", event.Before)
		}
		if event.PrimaryKey != nil {
			fmt.Printf("  主键: %+v\n", event.PrimaryKey)
		}
		
		if event.Operation != "DELETE" {
			t.Errorf("Expected DELETE operation, got %s", event.Operation)
		}
		if event.Table != "test_product" {
			t.Errorf("Expected test_product table, got %s", event.Table)
		}
		fmt.Println("✅ DELETE 测试通过")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for DELETE event")
	}
}
