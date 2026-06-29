import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { GoogleLogin } from "@react-oauth/google"
import { AlertCircle, MapPin } from "lucide-react"

import { useAuth } from "../../context/AuthContext"
import { getErrorMessage } from "../../services/api"

export function LoginPage() {
  const navigate = useNavigate()
  const { loginWithGoogle } = useAuth()

  const [error, setError] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  async function handleGoogleCredential(credential?: string) {
    if (!credential) {
      setError("Google did not return a sign-in credential.")
      return
    }

    setIsLoading(true)
    setError("")
    try {
      await loginWithGoogle(credential)
      navigate("/")
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-blue-950 to-slate-900 px-4">
      <div className="w-full max-w-md bg-white/5 backdrop-blur-xl border border-white/10 rounded-3xl shadow-2xl p-8">
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 bg-blue-500 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-blue-500/30">
            <MapPin size={24} className="text-white" />
          </div>
          <h1 className="text-2xl font-semibold text-white">ISOMAP</h1>
          <p className="text-sm text-white/50 mt-1">Journey-centred navigation</p>
        </div>

        {error && (
          <div className="flex items-start gap-2.5 bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 mb-4">
            <AlertCircle size={16} className="shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex flex-col gap-4">
          <div className={isLoading ? "pointer-events-none opacity-60" : ""}>
            <GoogleLogin
              onSuccess={(response) => handleGoogleCredential(response.credential)}
              onError={() => setError("Google sign-in failed.")}
              width="336"
            />
          </div>

          {isLoading && (
            <div className="flex items-center justify-center gap-2 text-sm text-white/50">
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeOpacity="0.25" />
                <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
              </svg>
              Signing in...
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
