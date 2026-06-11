import { Gift, Megaphone, PackageCheck, Send, ShieldAlert, type LucideIcon } from 'lucide-react'

export type MailType = 'all' | 'gm_notice' | 'compensation' | 'reward' | 'event_reward' | 'player_message'

export interface MockMail {
  id: string
  type: Exclude<MailType, 'all'>
  sender: string
  title: string
  content: string
  createdAt: string
  read: boolean
  hasAttachment: boolean
  claimed: boolean
  expiresAt?: string
}

export const PAGE_SIZE = 5

export const TYPE_OPTIONS: Array<{ key: MailType; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'gm_notice', label: 'GM 通知' },
  { key: 'compensation', label: '补偿' },
  { key: 'reward', label: '奖励' },
  { key: 'event_reward', label: '活动' },
  { key: 'player_message', label: '私信' },
]

export const TYPE_CONFIG: Record<Exclude<MailType, 'all'>, { label: string; icon: LucideIcon; color: string }> = {
  gm_notice: { label: 'GM 通知', icon: ShieldAlert, color: 'text-blue-600 bg-blue-500/10' },
  compensation: { label: '补偿', icon: Gift, color: 'text-amber-600 bg-amber-500/10' },
  reward: { label: '奖励', icon: PackageCheck, color: 'text-emerald-600 bg-emerald-500/10' },
  event_reward: { label: '活动', icon: Megaphone, color: 'text-purple-600 bg-purple-500/10' },
  player_message: { label: '私信', icon: Send, color: 'text-rose-600 bg-rose-500/10' },
}

export const MOCK_MAILS: MockMail[] = [
  {
    id: 'mail_001',
    type: 'gm_notice',
    sender: 'Hero3 GM',
    title: '信函系统第一版 UI 预览',
    content: '当前页面只实现信函系统 UI。后续接入后端后，这里会展示真实 GM 通知、系统补偿、活动奖励和玩家私信。',
    createdAt: '2026-06-11 18:40',
    read: false,
    hasAttachment: false,
    claimed: false,
  },
  {
    id: 'mail_002',
    type: 'compensation',
    sender: '系统',
    title: '维护补偿预留样式',
    content: '补偿类信函后续会支持资源、城金、金币、兵种、将领经验和限时加成附件。第一版 UI 先展示附件区域和领取状态。',
    createdAt: '2026-06-11 15:20',
    read: false,
    hasAttachment: true,
    claimed: false,
    expiresAt: '2026-07-11',
  },
  {
    id: 'mail_003',
    type: 'event_reward',
    sender: '活动使者',
    title: '活动奖励入口预留',
    content: '活动奖励后续会由活动模块生成，信函系统只负责投递、展示和领取。',
    createdAt: '2026-06-10 21:08',
    read: true,
    hasAttachment: true,
    claimed: true,
  },
  {
    id: 'mail_004',
    type: 'reward',
    sender: '系统',
    title: '成长奖励预留',
    content: '普通奖励可用于新手引导、阶段目标、特殊成就等模块。',
    createdAt: '2026-06-10 09:12',
    read: true,
    hasAttachment: true,
    claimed: false,
    expiresAt: '2026-07-10',
  },
  {
    id: 'mail_005',
    type: 'player_message',
    sender: '刘玄德',
    title: '玩家互发信函样式',
    content: '玩家互发属于第二阶段功能，正式开放前需要发送冷却、每日上限、屏蔽和举报机制。',
    createdAt: '2026-06-09 22:31',
    read: true,
    hasAttachment: false,
    claimed: false,
  },
  {
    id: 'mail_006',
    type: 'gm_notice',
    sender: 'Hero3 GM',
    title: '版本调整说明样式',
    content: 'GM 通知适合承载单个玩家相关的说明，不替代公告系统。',
    createdAt: '2026-06-08 13:45',
    read: true,
    hasAttachment: false,
    claimed: false,
  },
]
