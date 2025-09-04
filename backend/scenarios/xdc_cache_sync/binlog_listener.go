package xdccachesync

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/schema"
)

type BinlogListener struct {
	canal       *canal.Canal
	eventChan   chan *CDCEvent
	stopChan    chan struct{}
	running     bool
	mu          sync.RWMutex
	tableFilter map[string]bool // tables to monitor
}

func NewBinlogListener(cfg *canal.Config) (*BinlogListener, error) {
	c, err := canal.NewCanal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create canal: %w", err)
	}

	listener := &BinlogListener{
		canal:       c,
		eventChan:   make(chan *CDCEvent, 1000),
		stopChan:    make(chan struct{}),
		tableFilter: make(map[string]bool),
	}

	c.SetEventHandler(listener)

	return listener, nil
}

func (bl *BinlogListener) AddTableFilter(schema, table string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	key := fmt.Sprintf("%s.%s", schema, table)
	bl.tableFilter[key] = true
}

func (bl *BinlogListener) Start() error {
	bl.mu.Lock()
	if bl.running {
		bl.mu.Unlock()
		return fmt.Errorf("binlog listener is already running")
	}
	bl.running = true
	bl.mu.Unlock()

	go func() {
		defer func() {
			bl.mu.Lock()
			bl.running = false
			bl.mu.Unlock()
		}()

		pos := mysql.Position{Name: "", Pos: 4}
		err := bl.canal.RunFrom(pos)
		if err != nil {
			log.Printf("Canal run error: %v", err)
		}
	}()

	return nil
}

func (bl *BinlogListener) Stop() {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if !bl.running {
		return
	}

	close(bl.stopChan)
	bl.canal.Close()
	bl.running = false
}

func (bl *BinlogListener) GetEventChannel() <-chan *CDCEvent {
	return bl.eventChan
}

func (bl *BinlogListener) OnRotate(header *replication.EventHeader, rotateEvent *replication.RotateEvent) error {
	return nil
}

func (bl *BinlogListener) OnTableChanged(header *replication.EventHeader, schema string, table string) error {
	return nil
}

func (bl *BinlogListener) OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	return nil
}

// OnRow handles row change events (INSERT, UPDATE, DELETE)
func (bl *BinlogListener) OnRow(e *canal.RowsEvent) error {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	// Check if we should monitor this table
	tableKey := fmt.Sprintf("%s.%s", e.Table.Schema, e.Table.Name)
	if len(bl.tableFilter) > 0 && !bl.tableFilter[tableKey] {
		return nil
	}

	// Convert canal event to our CDC event format
	for i, row := range e.Rows {
		cdcEvent := &CDCEvent{
			Timestamp: time.Now(),
			Schema:    e.Table.Schema,
			Table:     e.Table.Name,
			Operation: e.Action,
		}

		// Extract primary key
		cdcEvent.PrimaryKey = make(map[string]interface{})
		for _, pkCol := range e.Table.PKColumns {
			if pkCol < len(row) {
				cdcEvent.PrimaryKey[e.Table.Columns[pkCol].Name] = row[pkCol]
			}
		}

		// Handle different event types
		switch e.Action {
		case canal.InsertAction:
			cdcEvent.Operation = "INSERT"
			cdcEvent.After = bl.rowToMap(e.Table, row)

		case canal.UpdateAction:
			cdcEvent.Operation = "UPDATE"
			if i%2 == 0 { // Before row
				cdcEvent.Before = bl.rowToMap(e.Table, row)
			} else { // After row
				cdcEvent.After = bl.rowToMap(e.Table, row)
				// Send the event only for the after row
				select {
				case bl.eventChan <- cdcEvent:
				case <-bl.stopChan:
					return nil
				default:
					log.Printf("Event channel is full, dropping event")
				}
			}
			continue

		case canal.DeleteAction:
			cdcEvent.Operation = "DELETE"
			cdcEvent.Before = bl.rowToMap(e.Table, row)
		}

		// Send event to channel (except for UPDATE which is handled above)
		if e.Action != canal.UpdateAction {
			select {
			case bl.eventChan <- cdcEvent:
			case <-bl.stopChan:
				return nil
			default:
				log.Printf("Event channel is full, dropping event")
			}
		}
	}

	return nil
}

// OnXID handles transaction commit events
func (bl *BinlogListener) OnXID(header *replication.EventHeader, nextPos mysql.Position) error {
	return nil
}

// OnGTID handles GTID events
func (bl *BinlogListener) OnGTID(header *replication.EventHeader, gtid mysql.BinlogGTIDEvent) error {
	return nil
}

// OnRowsQueryEvent handles rows query events
func (bl *BinlogListener) OnRowsQueryEvent(event *replication.RowsQueryEvent) error {
	return nil
}

// OnPosSynced handles position sync events
func (bl *BinlogListener) OnPosSynced(header *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) error {
	return nil
}

// String returns string representation
func (bl *BinlogListener) String() string {
	return "XDCCacheSyncBinlogListener"
}

// rowToMap converts a row to a map using column names
func (bl *BinlogListener) rowToMap(table *schema.Table, row []interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i, col := range table.Columns {
		if i < len(row) {
			result[col.Name] = row[i]
		}
	}
	return result
}

func CreateDefaultCanalConfig() *canal.Config {
	cfg := canal.NewDefaultConfig()
	cfg.Addr = "mysql:3306"
	cfg.User = "root"
	cfg.Password = "root"
	cfg.Charset = "utf8mb4"
	cfg.Flavor = "mysql"

	cfg.IncludeTableRegex = []string{"playground\\.web_product"}
	cfg.ExcludeTableRegex = []string{}

	// Use row-based replication
	cfg.Dump.ExecutionPath = ""
	cfg.Dump.DiscardErr = false

	return cfg
}
