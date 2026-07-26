import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { GoogleLogin } from "@react-oauth/google"
import { AlertCircle, LogIn, Navigation, Clock, Users } from "lucide-react"

import { useAuth } from "../../context/AuthContext"
import { getErrorMessage } from "../../services/api"
import logo from "../../src/assets/isomap_white.png"

export function LoginPage() {
  const navigate = useNavigate()
  const { loginWithGoogle, continueAsGuest } = useAuth()

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

  function handleGuest() {
    continueAsGuest()
    navigate("/")
  }

  return (
    <div className="min-h-screen flex bg-brand-950">

      {/* ── LEFT SIDE — Login form ───────────────────────────── */}
      <div 
        className="
          w-full md:flex-[9]
          flex-shrink-0 flex flex-col justify-center 
          px-6 sm:px-10 py-10 md:py-12 
          bg-brand-950 border-r border-white/5
        "
      >

        {/* Logo + name */}
        <div className="flex flex-col items-center gap-0 mb-10">
          <img
            src={logo}
            alt="ISOMAP logo"
            className="object-contain"
            style = {{
              width: "clamp(120px, 32vw, 400px)",
              height: "auto",
            }}
          />
          <p className="text-sm text-white/40 mt-1">Journey-centred navigation</p>
        </div>

        {/* Error banner */}
        {error && (
          <div className="flex items-start gap-2.5 bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 mb-4">
            <AlertCircle size={16} className="shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex flex-col gap-3">

          {/* Google login */}
          <div className={isLoading ? "pointer-events-none opacity-60" : ""}>
            <GoogleLogin
              onSuccess={(res) => handleGoogleCredential(res.credential)}
              onError={() => setError("Google sign-in failed.")}
              width="100%"
              theme="filled_white"
              shape="rectangular"
              size="large"
            />
          </div>

          {/* Loading indicator */}
          {isLoading && (
            <div className="flex items-center justify-center gap-2 text-sm text-white/40">
              <svg className="animate-spin h-4 w-4 text-brand-400" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeOpacity="0.25" />
                <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
              </svg>
              Signing in...
            </div>
          )}

          {/* Divider */}
          <div className="flex items-center gap-3 my-1">
            <div className="flex-1 h-px bg-white/10" />
            <span className="text-xs text-white/25">or</span>
            <div className="flex-1 h-px bg-white/10" />
          </div>

          {/* Guest button */}
          <button
            onClick={handleGuest}
            disabled={isLoading}
            className="w-full flex items-center justify-center gap-2 py-3
              bg-white/5 hover:bg-brand-800/40
              border border-white/10 hover:border-brand-400/30
              text-white/70 hover:text-white
              text-sm font-medium rounded-xl transition-all
              disabled:opacity-10 disabled:cursor-not-allowed"
          >
            <LogIn size={16} />
            Continue as Guest
          </button>

          {/* Guest limitation notice */}
          <p className="text-center text-xs text-white/20 leading-relaxed">
            Guest users have access to routing and proximity search.
            <br />
            Group planning requires a Google account.
          </p>

        </div>
      </div>

      {/* ── RIGHT SIDE — Feature showcase ───────────────────── */}
      <div 
        className="
          hidden md:flex flex-[11]
          flex-col justify-center 
          px-8 lg:px-16
          bg-gradient-to-br from-brand-900 via-brand-950 to-brand-950
        "
      >

        {/* Tagline */}
        <div className="mb-12">
          <h2 className="text-2xl lg:text-4xl font-bold text-white leading-tight mb-3">
            Navigate smarter.<br />
            <span className="text-brand-200">Travel better.</span>
          </h2>
          <p className="text-white/40 text-base max-w-sm leading-relaxed">
            More than the fastest route — ISOMAP helps you make the most of every journey.
          </p>
        </div>

        {/* Feature cards */}
        <div className="flex flex-col gap-5 max-w-sm">
          <FeatureItem
            icon={<Navigation size={20} />}
            title="Smart Routing"
            description="Routes that adapt to real-time traffic and crowd conditions across all transport modes."
          />
          <FeatureItem
            icon={<Clock size={20} />}
            title="Proximity Search"
            description="Discover places of interest within your own time window and category of choice."
          />
          <FeatureItem
            icon={<Users size={20} />}
            title="Group Meetup"
            description="Create a group with friends and find the ideal meeting point in a blink of an eye."
            guestLocked
          />
        </div>

      </div>
    </div>
  )
}

// ── Feature item component ────────────────────────────────────────────────────
function FeatureItem({
  icon,
  title,
  description,
  guestLocked = false,
}: {
  icon: React.ReactNode
  title: string
  description: string
  guestLocked?: boolean
}) {
  return (
    <div className="flex items-start gap-4">
      {/* Icon bubble */}
      <div className="w-10 h-10 rounded-xl bg-brand-600/30 border border-brand-400/20
        flex items-center justify-center text-brand-200 shrink-0 mt-0.5">
        {icon}
      </div>

      {/* Text */}
      <div>
        <div className="flex items-center gap-2 mb-0.5">
          <h3 className="text-sm font-semibold text-white">{title}</h3>
          {guestLocked && (
            <span className="text-[10px] text-brand-200/60 border border-brand-400/20
              rounded-full px-2 py-0.5 leading-none">
              Sign in required
            </span>
          )}
        </div>
        <p className="text-sm text-white/35 leading-relaxed">{description}</p>
      </div>
    </div>
  )
}