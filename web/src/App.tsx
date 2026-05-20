import { useState } from 'react'
import Layout from './components/Layout'
import CityPage from './pages/CityPage'
import MilitaryPage from './pages/MilitaryPage'
import MapPage from './pages/MapPage'
import ReportsPage from './pages/ReportsPage'
import './App.css'

function App() {
  const [activePage, setActivePage] = useState('city')

  const renderPage = () => {
    switch (activePage) {
      case 'city':
        return <CityPage />
      case 'military':
        return <MilitaryPage />
      case 'map':
        return <MapPage />
      case 'reports':
        return <ReportsPage />
      default:
        return <CityPage />
    }
  }

  return (
    <Layout activeKey={activePage} onNavigate={setActivePage}>
      {renderPage()}
    </Layout>
  )
}

export default App
