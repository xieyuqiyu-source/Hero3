import './App.css'
import { useState } from 'react'
import AccountsPanel from '@/components/AccountsPanel'
import AdminLayout, { type AdminPage } from '@/components/AdminLayout'
import ApiDiagnosticsPanel from '@/components/ApiDiagnosticsPanel'
import { AuditPanel, GuardrailPanel } from '@/components/AuditPanel'
import BalanceConfigPanel from '@/components/BalanceConfigPanel'
import MetricsGrid from '@/components/MetricsGrid'
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
          <div className="page-grid two-column">
            <ResourceToolsPanel />
            <SystemActionsPanel />
          </div>
        )
      case 'balance':
        return <BalanceConfigPanel />
      case 'api':
        return <ApiDiagnosticsPanel />
      case 'audit':
        return (
          <>
            <AuditPanel />
            <GuardrailPanel />
          </>
        )
      default:
        return (
          <>
            <MetricsGrid stats={dashboardStats} />
            <div className="page-grid two-column">
              <AccountsPanel
                accounts={accounts}
                busyTarget={busyTarget}
                gameState={gameState}
                onDeleteAccount={handleDeleteAccount}
                onDeletePlayer={handleDeletePlayer}
              />
              <div className="stacked-panels">
                <ResourceToolsPanel />
                <SystemActionsPanel />
              </div>
            </div>
          </>
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
      {error && <section className="status-banner">后端未连接：{error}</section>}
      {actionMessage && <section className="success-banner">{actionMessage}</section>}

      <section className="page-surface">{renderPage()}</section>
    </AdminLayout>
  )
}

export default App
