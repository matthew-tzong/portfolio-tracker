import { useState } from 'react'
import { supabase } from '../lib/supabase'
import { useNavigate } from 'react-router-dom'

// Authentication screen (sign-in only), UI never exposes public sign-up form.
export function Auth() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  // Sign-in handler for the auth form.
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const { error } = await supabase.auth.signInWithPassword({ email, password })
      if (error) {
        throw error
      }
      navigate('/dashboard', { replace: true })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-6">
      <div className="w-full max-w-md bg-card border border-border rounded-4xl p-10 shadow-2xl relative overflow-hidden group">
        <div className="mb-10 pt-4">
          <h1 className="text-3xl font-bold text-white tracking-tight mb-2">Welcome Back</h1>
          <p className="text-zinc-500 text-sm font-medium">
            Sign in to your private portfolio tracker.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label
              htmlFor="email"
              className="block text-xs font-bold text-zinc-500 uppercase tracking-widest mb-3 ml-1"
            >
              Email Address
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="name@example.com"
              className="w-full bg-zinc-900 border border-border text-zinc-100 rounded-2xl px-5 py-3.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary transition-all placeholder:text-zinc-700"
            />
          </div>
          <div>
            <label
              htmlFor="password"
              className="block text-xs font-bold text-zinc-500 uppercase tracking-widest mb-3 ml-1"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="••••••••"
              className="w-full bg-zinc-900 border border-border text-zinc-100 rounded-2xl px-5 py-3.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary transition-all placeholder:text-zinc-700"
            />
          </div>

          {error && (
            <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-2xl text-red-400 text-sm font-medium animate-shake">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full py-4 px-6 bg-primary text-background text-sm font-bold rounded-2xl hover:bg-green-400 transition-all shadow-xl active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed mt-4"
          >
            {loading ? (
              <div className="flex items-center justify-center gap-2">
                <span className="w-4 h-4 border-2 border-background/30 border-t-background rounded-full animate-spin" />
                <span>Authenticating...</span>
              </div>
            ) : (
              'Sign In to Dashboard'
            )}
          </button>
        </form>
      </div>
    </div>
  )
}
