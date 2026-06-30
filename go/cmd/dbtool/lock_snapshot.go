// 本文件提供 MySQL 锁等待和活跃事务的只读快照工具。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/infrastructure/storage"
)

type lockSnapshotOptions struct {
	Limit        int
	MinSeconds   int
	IncludeSleep bool
}

type lockSnapshot struct {
	DatabaseName string
	Processes    []processSnapshot
	Transactions []transactionSnapshot
	Waits        []lockWaitSnapshot
	Warnings     []string
}

type processSnapshot struct {
	ID      int64
	User    string
	Host    string
	DB      string
	Command string
	Seconds int
	State   string
	Info    string
}

type transactionSnapshot struct {
	ID       string
	State    string
	Seconds  int
	ThreadID int64
	Query    string
}

type lockWaitSnapshot struct {
	WaitingThread  int64
	BlockingThread int64
	WaitingEngine  string
	BlockingEngine string
	ObjectSchema   string
	ObjectName     string
	IndexName      string
	LockType       string
	LockMode       string
}

// runLockSnapshot 输出当前数据库连接、事务和锁等待快照。
func runLockSnapshot(args []string) error {
	flags := flag.NewFlagSet("lock-snapshot", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	limit := flags.Int("limit", 20, "每类最多输出多少条")
	minSeconds := flags.Int("min-seconds", 0, "只输出运行秒数不小于该值的连接或事务")
	includeSleep := flags.Bool("include-sleep", false, "是否包含 Sleep 连接")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveLockSnapshotDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	snapshot, err := collectLockSnapshot(ctx, resolvedDSN, lockSnapshotOptions{
		Limit:        *limit,
		MinSeconds:   *minSeconds,
		IncludeSleep: *includeSleep,
	})
	if err != nil {
		return err
	}
	snapshot.DatabaseName = databaseName
	printLockSnapshot(snapshot)
	return nil
}

// collectLockSnapshot 只读采集锁等待和活跃事务。
func collectLockSnapshot(ctx context.Context, dsn string, options lockSnapshotOptions) (lockSnapshot, error) {
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return lockSnapshot{}, err
	}
	defer db.Close()

	snapshot := lockSnapshot{}
	processes, err := queryProcessSnapshots(ctx, db, options)
	if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "processlist: "+err.Error())
	} else {
		snapshot.Processes = processes
	}
	transactions, err := queryTransactionSnapshots(ctx, db, options)
	if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "innodb_trx: "+err.Error())
	} else {
		snapshot.Transactions = transactions
	}
	waits, err := queryLockWaitSnapshots(ctx, db, options)
	if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "data_lock_waits: "+err.Error())
	} else {
		snapshot.Waits = waits
	}
	return snapshot, nil
}

// queryProcessSnapshots 读取当前活跃连接。
func queryProcessSnapshots(ctx context.Context, db *sql.DB, options lockSnapshotOptions) ([]processSnapshot, error) {
	includeSleep := 0
	if options.IncludeSleep {
		includeSleep = 1
	}
	rows, err := db.QueryContext(ctx,
		`SELECT ID, USER, HOST, COALESCE(DB, ''), COMMAND, TIME, COALESCE(STATE, ''), COALESCE(INFO, '')
		 FROM information_schema.PROCESSLIST
		 WHERE (? = 1 OR COMMAND <> 'Sleep') AND TIME >= ?
		 ORDER BY TIME DESC, ID ASC
		 LIMIT ?`,
		includeSleep,
		options.MinSeconds,
		options.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []processSnapshot{}
	for rows.Next() {
		var item processSnapshot
		if err := rows.Scan(&item.ID, &item.User, &item.Host, &item.DB, &item.Command, &item.Seconds, &item.State, &item.Info); err != nil {
			return nil, err
		}
		item.Info = compactSQL(item.Info)
		result = append(result, item)
	}
	return result, rows.Err()
}

// queryTransactionSnapshots 读取 InnoDB 活跃事务。
func queryTransactionSnapshots(ctx context.Context, db *sql.DB, options lockSnapshotOptions) ([]transactionSnapshot, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT trx_id, trx_state, TIMESTAMPDIFF(SECOND, trx_started, NOW()), trx_mysql_thread_id, COALESCE(trx_query, '')
		 FROM information_schema.INNODB_TRX
		 WHERE TIMESTAMPDIFF(SECOND, trx_started, NOW()) >= ?
		 ORDER BY trx_started ASC
		 LIMIT ?`,
		options.MinSeconds,
		options.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []transactionSnapshot{}
	for rows.Next() {
		var item transactionSnapshot
		if err := rows.Scan(&item.ID, &item.State, &item.Seconds, &item.ThreadID, &item.Query); err != nil {
			return nil, err
		}
		item.Query = compactSQL(item.Query)
		result = append(result, item)
	}
	return result, rows.Err()
}

// queryLockWaitSnapshots 读取 performance_schema 中的锁等待关系。
func queryLockWaitSnapshots(ctx context.Context, db *sql.DB, options lockSnapshotOptions) ([]lockWaitSnapshot, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT
			COALESCE(waiting_trx.PROCESSLIST_ID, 0),
			COALESCE(blocking_trx.PROCESSLIST_ID, 0),
			COALESCE(waiting_lock.ENGINE_LOCK_ID, ''),
			COALESCE(blocking_lock.ENGINE_LOCK_ID, ''),
			COALESCE(waiting_lock.OBJECT_SCHEMA, ''),
			COALESCE(waiting_lock.OBJECT_NAME, ''),
			COALESCE(waiting_lock.INDEX_NAME, ''),
			COALESCE(waiting_lock.LOCK_TYPE, ''),
			COALESCE(waiting_lock.LOCK_MODE, '')
		 FROM performance_schema.data_lock_waits waits
		 JOIN performance_schema.data_locks waiting_lock
		   ON waiting_lock.ENGINE_LOCK_ID = waits.REQUESTING_ENGINE_LOCK_ID
		 JOIN performance_schema.data_locks blocking_lock
		   ON blocking_lock.ENGINE_LOCK_ID = waits.BLOCKING_ENGINE_LOCK_ID
		 LEFT JOIN performance_schema.threads waiting_trx
		   ON waiting_trx.THREAD_ID = waiting_lock.THREAD_ID
		 LEFT JOIN performance_schema.threads blocking_trx
		   ON blocking_trx.THREAD_ID = blocking_lock.THREAD_ID
		 LIMIT ?`,
		options.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []lockWaitSnapshot{}
	for rows.Next() {
		var item lockWaitSnapshot
		if err := rows.Scan(
			&item.WaitingThread,
			&item.BlockingThread,
			&item.WaitingEngine,
			&item.BlockingEngine,
			&item.ObjectSchema,
			&item.ObjectName,
			&item.IndexName,
			&item.LockType,
			&item.LockMode,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// resolveLockSnapshotDSN 解析锁快照 DSN。
func resolveLockSnapshotDSN(rawDSN string) (string, string, error) {
	dsn, err := resolveBattleReportDSN(rawDSN)
	if err != nil {
		return "", "", err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return "", "", err
	}
	return dsn, databaseName, nil
}

// printLockSnapshot 输出锁快照。
func printLockSnapshot(snapshot lockSnapshot) {
	fmt.Printf("数据库: %s\n", snapshot.DatabaseName)
	if len(snapshot.Warnings) > 0 {
		fmt.Println("警告:")
		for _, warning := range snapshot.Warnings {
			fmt.Printf("  %s\n", warning)
		}
		fmt.Println("")
	}
	fmt.Println("活跃连接:")
	for _, item := range snapshot.Processes {
		fmt.Printf("  id=%d user=%s db=%s cmd=%s seconds=%d state=%s sql=%s\n", item.ID, item.User, item.DB, item.Command, item.Seconds, item.State, item.Info)
	}
	if len(snapshot.Processes) == 0 {
		fmt.Println("  无")
	}
	fmt.Println("")
	fmt.Println("InnoDB 事务:")
	for _, item := range snapshot.Transactions {
		fmt.Printf("  trx=%s state=%s thread=%d seconds=%d sql=%s\n", item.ID, item.State, item.ThreadID, item.Seconds, item.Query)
	}
	if len(snapshot.Transactions) == 0 {
		fmt.Println("  无")
	}
	fmt.Println("")
	fmt.Println("锁等待:")
	for _, item := range snapshot.Waits {
		fmt.Printf("  waiting_thread=%d blocking_thread=%d object=%s.%s index=%s type=%s mode=%s\n", item.WaitingThread, item.BlockingThread, item.ObjectSchema, item.ObjectName, item.IndexName, item.LockType, item.LockMode)
	}
	if len(snapshot.Waits) == 0 {
		fmt.Println("  无")
	}
}

// compactSQL 压缩 SQL 空白并限制输出长度。
func compactSQL(sqlText string) string {
	sqlText = strings.Join(strings.Fields(sqlText), " ")
	if len(sqlText) > 240 {
		return sqlText[:240] + "..."
	}
	return sqlText
}
