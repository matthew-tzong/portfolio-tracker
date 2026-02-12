import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Auth } from './components/Auth'
import { Dashboard } from './components/Dashboard'
import { ProtectedRoute } from './components/ProtectedRoute'
import { LinkManagement } from './components/LinkManagement'

/**
Route graph:
 - `/auth`: sign in / create account
 - `/dashboard`: protected main app shell
 - `/`: redirects to `/dashboard` (which will bounce unauthenticated users to `/auth`)
 - `/links`: protected link management page
 */
function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/auth" element={<Auth />} />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />
        <Route
          path="/links"
          element={
            <ProtectedRoute>
              <LinkManagement />
            </ProtectedRoute>
          }
        />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
