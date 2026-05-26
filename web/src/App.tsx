import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { useEffect } from 'react'
import Layout from './components/Layout'
import RequirePlayer from './components/RequirePlayer'
import ChangelogModal from './components/ChangelogModal'
import { ToastContainer } from './components/ui'
import { useConfigStore } from './store/configStore'
import LoginPage from './pages/login'
import ReportSharePage from './pages/report/ReportSharePage'
import CityPage from './pages/city'
import MilitaryPage from './pages/military'
import MapPage from './pages/map'
import NewsPage from './pages/news'

import AccountPage from './pages/account'
import SettingsPage from './pages/settings'
import './App.css'

function GameLayout() {
  return (
    <Layout>
      <Outlet />
    </Layout>
  )
}

function App() {
  const loadBootstrap = useConfigStore((s) => s.loadBootstrap)

  useEffect(() => {
    loadBootstrap()
  }, [loadBootstrap])

  return (
    <>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/report/:reportId" element={<ReportSharePage />} />
        {/* All game routes require an active player */}
        <Route element={<RequirePlayer />}>
          <Route element={<GameLayout />}>
            <Route path="/" element={<Navigate to="/city" replace />} />
            <Route path="/city" element={<CityPage />} />
            <Route path="/military" element={<MilitaryPage />} />
            <Route path="/map" element={<MapPage />} />
            <Route path="/news" element={<NewsPage />} />

            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/account" element={<AccountPage />} />
          </Route>
        </Route>
      </Routes>
      <ToastContainer />
      <ChangelogModal />
    </>
  )
}

export default App
