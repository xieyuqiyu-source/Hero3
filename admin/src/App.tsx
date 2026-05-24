import './App.css'
import { useState } from 'react'
import AccountsPanel from '@/components/AccountsPanel'
import AdminLayout, { type AdminPage } from '@/components/AdminLayout'
import ApiDiagnosticsPanel from '@/components/ApiDiagnosticsPanel'
import { AuditPanel, GuardrailPanel } from '@/components/AuditPanel'
import BalanceConfigPanel from '@/components/BalanceConfigPanel'
import CombatConfigPanel from '@/components/CombatConfigPanel'
import FactionsConfigPanel from '@/components/FactionsConfigPanel'
import MetricsGrid from '@/components/MetricsGrid'
import NpcConfigPanel from '@/components/NpcConfigPanel'
import UnitsConfigPanel from '@/components/UnitsConfigPanel'
import { ResourceToolsPanel, SystemActionsPanel } from '@/components/OperationsPanel'
import { useAdminDashboard } from '@/hooks/useAdminDashboard'
import type { AccountSummary, PlayerSummary } from '@/types'

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
    gameState,
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
            gameState={gameState}
            onDeleteAccount={handleDeleteAccount}
            onDeletePlayer={handleDeletePlayer}
          />
        )
      case 'operations':
        return (
          <div className="grid gap-4 lg:grid-cols-2">
            <ResourceToolsPanel />
            <SystemActionsPanel />
          </div>
        )
      case 'balance':
        return (
          <div className="grid gap-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <BalanceConfigPanel />
              <NpcConfigPanel />
            </div>
            <div className="grid gap-4 lg:grid-cols-2">
              <CombatConfigPanel />
              <FactionsConfigPanel />
            </div>
            <UnitsConfigPanel />
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
          <div className="grid gap-4">
            <MetricsGrid stats={dashboardStats} />
            <div className="grid gap-4 lg:grid-cols-[1.3fr_0.7fr]">
              <AccountsPanel
                accounts={accounts}
                busyTarget={busyTarget}
                gameState={gameState}
                onDeleteAccount={handleDeleteAccount}
                onDeletePlayer={handleDeletePlayer}
              />
              <div className="grid gap-4 content-start">
                <ResourceToolsPanel />
                <SystemActionsPanel />
              </div>
            </div>
          </div>
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
