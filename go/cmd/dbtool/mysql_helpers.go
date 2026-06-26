// 本文件归口数据库维护工具的 MySQL 标识符和切片辅助函数。
package main

import "strings"

// joinQuotedIdentifiers 拼接转义后的标识符。
func joinQuotedIdentifiers(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteIdentifier(value))
	}
	return strings.Join(quoted, ",")
}

// quoteIdentifier 转义 MySQL 标识符。
func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}
