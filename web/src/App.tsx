import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import Layout from './components/Layout'
import { ToastContainer } from './components/ui'
import LoginPage from './pages/LoginPage'
import CityPage from './pages/CityPage'
import MilitaryPage from './pages/MilitaryPage'
import MapPage from './pages/MapPage'
import ReportsPage from './pages/ReportsPage'
import SettingsPage from './pages/SettingsPage'
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
        <Route element={<GameLayout />}>
          <Route path="/" element={<Navigate to="/city" replace />} />
          <Route path="/city" element={<CityPage />} />
          <Route path="/military" element={<MilitaryPage />} />
          <Route path="/map" element={<MapPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
      <ToastContainer />
    </>
  )
}

export default App
