import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { supabase } from '../lib/supabase'

interface ProtectedRouteProps {
  children: React.ReactNode
}

type AuthStatus = 'checking' | 'authenticated' | 'unauthenticated'

// Authentication wrapper for routes that require a Supabase session.
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [status, setStatus] = useState<AuthStatus>('checking')

  useEffect(() => {
    let isMounted = true

    // Initialize the session status
    const initializeSession = async () => {
      const {data: { session }} = await supabase.auth.getSession()
      if (!isMounted) {
        return
      }
      setStatus(session ? 'authenticated' : 'unauthenticated')
    }

    initializeSession()
    const {data: { subscription }} = supabase.auth.onAuthStateChange((_event, session) => {
      if (!isMounted) {
        return
      }
      setStatus(session ? 'authenticated' : 'unauthenticated')
    })

    // Cleanup the subscription
    return () => {
      isMounted = false
      subscription.unsubscribe()
    }
  }, [])

  if (status === 'checking') {
    return (
      <div className="text-center mt-12 text-gray-600">
        <p>Loading...</p>
      </div>
    )
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/auth" replace />
  }

  return <>{children}</>
}

