import { Outlet } from 'react-router-dom'
import { AppHeader } from './AppHeader'

// Main layout for the app, shows the app header and the main content.
export function MainLayout() {
  return (
    <div className="min-h-screen bg-background">
      <AppHeader />
      <main className="pb-12">
        <Outlet />
      </main>
    </div>
  )
}
