// 本文件集中管理前端账号级可见性规则。

export const INTERNAL_TOOL_ACCOUNT = 'xieyuqi'

/** 判断当前账号是否可以查看内部工具入口。 */
export function canViewInternalTools(username?: string | null): boolean {
  return (username ?? '').trim().toLowerCase() === INTERNAL_TOOL_ACCOUNT
}
