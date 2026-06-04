import './App.css'
import { useState } from 'react'
import AccountsPanel from '@/components/AccountsPanel'
import AdminLayout, { type AdminPage } from '@/components/AdminLayout'
import ApiDiagnosticsPanel from '@/components/ApiDiagnosticsPanel'
import { AuditPanel, GuardrailPanel } from '@/components/AuditPanel'
import BalanceConfigPanel from '@/components/BalanceConfigPanel'
import CollapsiblePanel from '@/components/CollapsiblePanel'
import CombatConfigPanel from '@/components/CombatConfigPanel'
import FactionsConfigPanel from '@/components/FactionsConfigPanel'
import NpcConfigPanel from '@/components/NpcConfigPanel'
import UnitsConfigPanel from '@/components/UnitsConfigPanel'
import OverviewPage from '@/components/OverviewPage'
import { ResourceToolsPanel, SystemActionsPanel } from '@/components/OperationsPanel'
import BuffPanel from '@/components/BuffPanel'
import MiniGameRecordsPanel from '@/components/MiniGameRecordsPanel'
import AdminTokenPanel from '@/components/AdminTokenPanel'
import GeneralsConfigPanel from '@/components/GeneralsConfigPanel'
import { useAdminDashboard } from '@/hooks/useAdminDashboard'
import type { AccountSummary, PlayerSummary } from '@/types'
import { Sliders, MapPin, Swords, Flag, Shield, Users } from 'lucide-react'

function App() {
  const [activePage, setActivePage] = useState<AdminPage>('overview')
  const {
    accounts,
    actionMessage,
    busyTarget,
    dashboardStats,
    deleteAccount,
    deletePlayer,
    error,
    reload,
    health,
    loading,
  } = useAdminDashboard()

  const handleDeletePlayer = async (player: PlayerSummary) => {
    const confirmed = window.confirm(`确认删除云存档「${player.nickname}」？此操作不可恢复。`)
    if (!confirmed) return
    await deletePlayer(player.id)
  }

  const handleDeleteAccount = async (account: AccountSummary) => {
    const confirmed = window.confirm(`确认删除账号「${account.username}」及其 ${account.players.length} 个云存档？此操作不可恢复。`)
    if (!confirmed) return
    await deleteAccount(account.id)
  }

  const renderPage = () => {
    switch (activePage) {
      case 'accounts':
        return (
          <AccountsPanel
            accounts={accounts}
            busyTarget={busyTarget}
            onDeleteAccount={handleDeleteAccount}
            onDeletePlayer={handleDeletePlayer}
          />
        )
      case 'overview':
        return (
          <OverviewPage
            accounts={accounts}
            busyTarget={busyTarget}
            dashboardStats={dashboardStats}
            health={health}
            loading={loading}
            onDeletePlayer={handleDeletePlayer}
            onReload={reload}
          />
        )
      case 'operations':
        return (
          <div className="grid gap-4 lg:grid-cols-2">
            <AdminTokenPanel />
            <ResourceToolsPanel />
            <BuffPanel />
            <MiniGameRecordsPanel />
            <SystemActionsPanel />
          </div>
        )
      case 'balance':
        return (
          <div className="grid gap-4">
            <CollapsiblePanel icon={<Sliders size={16} className="text-[var(--color-accent)]" />} title="建筑数值">
              <BalanceConfigPanel />
            </CollapsiblePanel>
            <CollapsiblePanel icon={<MapPin size={16} className="text-[var(--color-accent)]" />} title="NPC 城池">
              <NpcConfigPanel />
            </CollapsiblePanel>
            <CollapsiblePanel icon={<Swords size={16} className="text-[var(--color-accent)]" />} title="战斗规则">
              <CombatConfigPanel />
            </CollapsiblePanel>
            <CollapsiblePanel icon={<Flag size={16} className="text-[var(--color-accent)]" />} title="阵营配置">
              <FactionsConfigPanel />
            </CollapsiblePanel>
            <CollapsiblePanel icon={<Shield size={16} className="text-[var(--color-accent)]" />} title="兵种配置">
              <UnitsConfigPanel />
            </CollapsiblePanel>
            <CollapsiblePanel icon={<Users size={16} className="text-[var(--color-accent)]" />} title="将领配置">
              <GeneralsConfigPanel />
            </CollapsiblePanel>
          </div>
        )
      case 'api':
        return <ApiDiagnosticsPanel />
      case 'audit':
        return (
          <div className="grid gap-4">
            <AuditPanel />
            <GuardrailPanel />
          </div>
        )
      default:
        return (
          <OverviewPage
            accounts={accounts}
            busyTarget={busyTarget}
            dashboardStats={dashboardStats}
            health={health}
            loading={loading}
            onDeletePlayer={handleDeletePlayer}
            onReload={reload}
          />
        )
    }
  }

  return (
    <AdminLayout
      activePage={activePage}
      environment={health?.environment ?? 'Offline'}
      loading={loading}
      onNavigate={setActivePage}
    >
      {error && (
        <div className="mb-4 px-4 py-3 rounded-2xl border border-red-500/30 bg-red-500/8 text-red-600 text-sm font-medium">
          后端未连接：{error}
        </div>
      )}
      {actionMessage && (
        <div className="mb-4 px-4 py-3 rounded-2xl border border-emerald-500/30 bg-emerald-500/8 text-emerald-700 text-sm font-medium">
          {actionMessage}
        </div>
      )}
      {renderPage()}
    </AdminLayout>
  )
}

export default App
