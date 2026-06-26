// Package main 提供 Hero3 数据库维护命令。
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// 自动加载 go/.env，方便本地开发直接复用服务配置。
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "create-test-db":
		err = runCreateTestDB(os.Args[2:])
	case "migrate-test":
		err = runMigrateTest(os.Args[2:])
	case "print-test-dsn":
		err = runPrintTestDSN(os.Args[2:])
	case "clone-data":
		err = runCloneData(os.Args[2:])
	case "backfill-resources":
		err = runBackfillResources(os.Args[2:])
	case "verify-resources":
		err = runVerifyResources(os.Args[2:])
	case "backfill-inventory":
		err = runBackfillInventory(os.Args[2:])
	case "verify-inventory":
		err = runVerifyInventory(os.Args[2:])
	case "backfill-buildings":
		err = runBackfillBuildings(os.Args[2:])
	case "verify-buildings":
		err = runVerifyBuildings(os.Args[2:])
	case "backfill-resource-slots":
		err = runBackfillResourceSlots(os.Args[2:])
	case "verify-resource-slots":
		err = runVerifyResourceSlots(os.Args[2:])
	case "backfill-army":
		err = runBackfillArmy(os.Args[2:])
	case "verify-army":
		err = runVerifyArmy(os.Args[2:])
	case "backfill-recruit-queues":
		err = runBackfillRecruitQueues(os.Args[2:])
	case "verify-recruit-queues":
		err = runVerifyRecruitQueues(os.Args[2:])
	case "backfill-generals":
		err = runBackfillGenerals(os.Args[2:])
	case "verify-generals":
		err = runVerifyGenerals(os.Args[2:])
	case "backfill-buffs":
		err = runBackfillBuffs(os.Args[2:])
	case "verify-buffs":
		err = runVerifyBuffs(os.Args[2:])
	default:
		printUsage()
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

// printUsage 输出数据库工具用法。
func printUsage() {
	fmt.Fprintln(os.Stderr, "用法：go run ./cmd/dbtool <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "命令：")
	fmt.Fprintln(os.Stderr, "  migrate            迁移当前 HERO3_DATABASE_DSN 数据库")
	fmt.Fprintln(os.Stderr, "  create-test-db     创建当前库对应的 test_ 前缀数据库")
	fmt.Fprintln(os.Stderr, "  migrate-test       创建并迁移当前库对应的 test_ 前缀数据库")
	fmt.Fprintln(os.Stderr, "  print-test-dsn     输出当前 DSN 对应的 test_ 前缀库 DSN")
	fmt.Fprintln(os.Stderr, "  clone-data         从源库复制数据到目标 test_ 库")
	fmt.Fprintln(os.Stderr, "  backfill-resources 从 state_json 兼容快照回填 player_resources")
	fmt.Fprintln(os.Stderr, "  verify-resources   校验 player_resources 与 state_json.resources 兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-inventory 从 state_json 兼容快照回填 player_inventory")
	fmt.Fprintln(os.Stderr, "  verify-inventory   校验 player_inventory 与 state_json.inventory 兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-buildings 从 state_json 兼容快照回填 player_buildings")
	fmt.Fprintln(os.Stderr, "  verify-buildings   校验 player_buildings 与 state_json.buildings 兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-resource-slots 从建筑快照回填 player_resource_slots")
	fmt.Fprintln(os.Stderr, "  verify-resource-slots   校验 player_resource_slots 与兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-army 从 state_json 兼容快照回填 player_army_units")
	fmt.Fprintln(os.Stderr, "  verify-army   校验 player_army_units 与 state_json.army 兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-recruit-queues 从 state_json 兼容快照回填 player_recruit_queues")
	fmt.Fprintln(os.Stderr, "  verify-recruit-queues   校验 player_recruit_queues 与 state_json.recruitQueues 兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-generals 从 state_json 兼容快照回填 player_generals 和 player_general_assignments")
	fmt.Fprintln(os.Stderr, "  verify-generals   校验 player_generals / player_general_assignments 与兼容快照")
	fmt.Fprintln(os.Stderr, "  backfill-buffs 从 state_json 兼容快照回填 player_buffs")
	fmt.Fprintln(os.Stderr, "  verify-buffs   校验 player_buffs 与 state_json.buffs 兼容快照")
}
