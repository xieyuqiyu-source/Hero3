import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import { ToastContainer } from './components/ui'
import CityPage from './pages/CityPage'
import MilitaryPage from './pages/MilitaryPage'
import MapPage from './pages/MapPage'
import ReportsPage from './pages/ReportsPage'
import SettingsPage from './pages/SettingsPage'
import './App.css'

function App() {
  return (
    <>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/city" replace />} />
          <Route path="/city" element={<CityPage />} />
          <Route path="/military" element={<MilitaryPage />} />
          <Route path="/map" element={<MapPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </Layout>
      <ToastContainer />
    </>
  )
}

export default App
