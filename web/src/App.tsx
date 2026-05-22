import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import Layout from './components/Layout'
import RequirePlayer from './components/RequirePlayer'
import { ToastContainer } from './components/ui'
import LoginPage from './pages/login'
import CityPage from './pages/city'
import MilitaryPage from './pages/military'
import MapPage from './pages/map'
import ReportsPage from './pages/reports'
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
  return (
    <>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        {/* All game routes require an active player */}
        <Route element={<RequirePlayer />}>
          <Route element={<GameLayout />}>
            <Route path="/" element={<Navigate to="/city" replace />} />
            <Route path="/city" element={<CityPage />} />
            <Route path="/military" element={<MilitaryPage />} />
            <Route path="/map" element={<MapPage />} />
            <Route path="/reports" element={<ReportsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/account" element={<AccountPage />} />
          </Route>
        </Route>
      </Routes>
      <ToastContainer />
    </>
  )
}

export default App
