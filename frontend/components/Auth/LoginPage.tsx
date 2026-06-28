// frontend/components/Auth/LoginPage.tsx
import { useState, type FormEvent } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../../context/AuthContext"
import { getErrorMessage } from "../../services/api"
import { MapPin, User, AlertCircle } from "lucide-react"

type Mode = "login" | "register"

export function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [mode, setMode] = useState<Mode>("login")
  const [username, setUsername] = useState("")
  const [error, setError] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (username.trim().length < 3) {
      setError("Username must be at least 3 characters.")
      return
    }
    setIsLoading(true)
    setError("")
    try {
      await login(username.trim())
      navigate("/")                    // goes to landing page
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-blue-950 to-slate-900 px-4">
      <div className="w-full max-w-md bg-white/5 backdrop-blur-xl border border-white/10 rounded-3xl shadow-2xl p-8">

        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 bg-blue-500 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-blue-500/30">
            <MapPin size={24} className="text-white" />
          </div>
          <h1 className="text-2xl font-semibold text-white">ISOMAP</h1>
          <p className="text-sm text-white/50 mt-1">Journey-centred navigation</p>
        </div>

        {/* Tab toggle */}
        <div className="flex bg-white/5 rounded-xl p-1 mb-6 border border-white/10">
          {(["login", "register"] as Mode[]).map((m) => (
            <button key={m} type="button" onClick={() => { setMode(m); setError("") }}
              className={`flex-1 py-2 text-sm font-medium rounded-lg transition-all duration-200 ${
                mode === m ? "bg-blue-500 text-white shadow-md" : "text-white/50 hover:text-white/80"
              }`}>
              {m === "login" ? "Sign in" : "Sign up"}
            </button>
          ))}
        </div>

        {/* Hint for dev */}
        <div className="bg-blue-500/10 border border-blue-500/20 text-blue-300 text-xs rounded-xl px-4 py-2.5 mb-4">
          💡 Use username <strong>Tester</strong> to sign in during development
        </div>

        {error && (
          <div className="flex items-start gap-2.5 bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 mb-4">
            <AlertCircle size={16} className="shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="flex items-center gap-3 bg-white/5 border border-white/10 rounded-xl px-4 py-3 focus-within:border-blue-500/60 transition-all">
            <User size={16} className="text-white/30 shrink-0" />
            <input
              type="text"
              placeholder="Enter your username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isLoading}
              autoComplete="username"
              className="flex-1 bg-transparent text-sm text-white placeholder-white/25 outline-none disabled:opacity-50"
            />
          </div>

          <button type="submit" disabled={isLoading}
            className="mt-1 w-full py-3 bg-blue-500 hover:bg-blue-400 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-xl transition-all active:scale-[0.98] shadow-lg shadow-blue-500/20">
            {isLoading ? (
              <span className="flex items-center justify-center gap-2">
                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeOpacity="0.25" />
                  <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
                </svg>
                {mode === "login" ? "Signing in…" : "Creating account…"}
              </span>
            ) : mode === "login" ? "Sign in" : "Create account"}
          </button>
        </form>
      </div>
    </div>
  )
}