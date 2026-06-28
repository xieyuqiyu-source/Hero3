import { Gift, Megaphone, PackageCheck, Radio, Send, ShieldAlert, Trophy, type LucideIcon } from 'lucide-react'
import type { Mail } from '@/types/game'

export type MailType = 'all' | Mail['mailType']
export type MailTypeConfig = { label: string; icon: LucideIcon; color: string }

export const PAGE_SIZE = 10

export const TYPE_OPTIONS: Array<{ key: MailType; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'gm_notice', label: 'GM 通知' },
  { key: 'compensation', label: '补偿' },
  { key: 'reward', label: '奖励' },
  { key: 'event_reward', label: '活动' },
  { key: 'pvp_season_reward', label: '赛季' },
  { key: 'server_broadcast', label: '喊话' },
  { key: 'system_notice', label: '系统' },
  { key: 'player_message', label: '私信' },
]

export const TYPE_CONFIG: Record<string, MailTypeConfig> = {
  gm_notice: { label: 'GM 通知', icon: ShieldAlert, color: 'text-blue-600 bg-blue-500/10' },
  compensation: { label: '补偿', icon: Gift, color: 'text-amber-600 bg-amber-500/10' },
  reward: { label: '奖励', icon: PackageCheck, color: 'text-emerald-600 bg-emerald-500/10' },
  event_reward: { label: '活动', icon: Megaphone, color: 'text-purple-600 bg-purple-500/10' },
  pvp_season_reward: { label: '赛季奖励', icon: Trophy, color: 'text-orange-600 bg-orange-500/10' },
  server_broadcast: { label: '全服喊话', icon: Radio, color: 'text-red-600 bg-red-500/10' },
  system_notice: { label: '系统', icon: Megaphone, color: 'text-sky-600 bg-sky-500/10' },
  player_message: { label: '私信', icon: Send, color: 'text-rose-600 bg-rose-500/10' },
}

export const DEFAULT_TYPE_CONFIG: MailTypeConfig = {
  label: '系统信函',
  icon: Megaphone,
  color: 'text-slate-600 bg-slate-500/10',
}

export function getMailTypeConfig(mailType: string): MailTypeConfig {
  return TYPE_CONFIG[mailType] ?? DEFAULT_TYPE_CONFIG
}
