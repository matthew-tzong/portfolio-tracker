import { useEffect, useState } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { supabase } from '../lib/supabase'

// CSS classes for the navigation links.
const navLinkClass =
  'py-2 px-3 text-sm font-medium rounded-md border focus:outline-none focus:ring-2 focus:ring-offset-2 min-w-[8.5rem] text-center inline-block'

const navLinkDefault = 'text-gray-700 bg-white border-gray-300 hover:bg-gray-50 focus:ring-gray-400'

const navLinkActive = 'text-blue-700 bg-blue-50 border-blue-200 ring-blue-400'

export function AppHeader() {
  const [user, setUser] = useState<{ email?: string } | null>(null)

  // Loads the authenticated user.
  useEffect(() => {
    supabase.auth.getUser().then(({ data: { user } }) => setUser(user))
  }, [])

  // Handles the sign out button click.
  const handleSignOut = async () => {
    await supabase.auth.signOut()
  }

  // Returns the app header.
  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-14">
          <Link to="/dashboard" className="text-xl font-semibold text-gray-900 hover:text-gray-700">
            My portfolio
          </Link>

          <nav className="flex items-center gap-2">
            <NavLink
              to="/dashboard"
              className={({ isActive: active }) =>
                `${navLinkClass} ${active ? navLinkActive : navLinkDefault}`
              }
            >
              Dashboard
            </NavLink>
            <NavLink
              to="/expenses"
              className={({ isActive: active }) =>
                `${navLinkClass} ${active ? navLinkActive : navLinkDefault}`
              }
            >
              Expense tracker
            </NavLink>
            <NavLink
              to="/budget"
              className={({ isActive: active }) =>
                `${navLinkClass} ${active ? navLinkActive : navLinkDefault}`
              }
            >
              Budget tracker
            </NavLink>
            <NavLink
              to="/portfolio"
              className={({ isActive: active }) =>
                `${navLinkClass} ${active ? navLinkActive : navLinkDefault}`
              }
            >
              Portfolio
            </NavLink>
            <NavLink
              to="/links"
              className={({ isActive: active }) =>
                `${navLinkClass} ${active ? navLinkActive : navLinkDefault}`
              }
            >
              Connections
            </NavLink>
            <button
              type="button"
              onClick={handleSignOut}
              className="py-2 px-4 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
            >
              Sign out
            </button>
          </nav>
        </div>
        {user?.email && (
          <p className="text-xs text-gray-500 pb-2 -mt-1">Signed in as {user.email}</p>
        )}
      </div>
    </header>
  )
}
