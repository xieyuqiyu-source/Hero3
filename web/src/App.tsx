import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import CityPage from './pages/CityPage'
import MilitaryPage from './pages/MilitaryPage'
import MapPage from './pages/MapPage'
import ReportsPage from './pages/ReportsPage'
import './App.css'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/city" replace />} />
        <Route path="/city" element={<CityPage />} />
        <Route path="/military" element={<MilitaryPage />} />
        <Route path="/map" element={<MapPage />} />
        <Route path="/reports" element={<ReportsPage />} />
      </Routes>
    </Layout>
  )
}

export default App
