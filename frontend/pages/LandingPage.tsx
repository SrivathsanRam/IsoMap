import type { ReactNode } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../context/AuthContext"
import {
  MapPin, Navigation, Clock, Users, Compass,
  Search, ChevronRight, LogOut, Zap, Map
} from "lucide-react"

export default function LandingPage() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <div className="min-h-screen bg-slate-950 text-white">

      {/* ── Top navigation bar ── */}
      <header className="flex items-center justify-between px-6 py-4 border-b border-white/5">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-blue-500 rounded-xl flex items-center justify-center shadow-lg shadow-blue-500/30">
            <MapPin size={16} className="text-white" />
          </div>
          <span className="font-semibold text-lg tracking-tight">ISOMAP</span>
        </div>

        <nav className="hidden md:flex items-center gap-6 text-sm text-white/50">
          <button onClick={() => navigate("/map")}
            className="hover:text-white transition-colors">Map</button>
          <button className="hover:text-white transition-colors">Proximity</button>
          <button className="hover:text-white transition-colors">Group</button>
        </nav>

        <div className="flex items-center gap-3">
          <span className="text-sm text-white/40 hidden sm:block">
            Hey, <span className="text-white/70">{user?.name}</span>
          </span>
          <button onClick={handleLogout}
            className="flex items-center gap-1.5 text-sm text-white/40 hover:text-red-400 transition-colors">
            <LogOut size={15} />
            <span className="hidden sm:block">Sign out</span>
          </button>
        </div>
      </header>

      {/* ── Hero ────────────────────────────────────────────────── */}
      <section className="relative flex flex-col items-center text-center px-6 pt-20 pb-16 overflow-hidden">
        {/* Glow blobs */}
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[600px] h-[400px] bg-blue-600/10 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute top-20 left-1/3 w-[300px] h-[300px] bg-indigo-600/8 rounded-full blur-3xl pointer-events-none" />

        <div className="relative z-10 flex flex-col items-center gap-6 max-w-2xl">
          <span className="inline-flex items-center gap-2 text-xs font-medium bg-blue-500/10 border border-blue-500/20 text-blue-400 rounded-full px-4 py-1.5">
            <Zap size={12} />
            Journey-centred navigation
          </span>

          <h1 className="text-5xl sm:text-6xl font-semibold leading-tight tracking-tight">
            Navigate smarter.<br />
            <span className="text-blue-400">Travel better.</span>
          </h1>

          <p className="text-white/50 text-lg max-w-lg leading-relaxed">
            More than the fastest route — discover what you can reach, coordinate with friends,
            and make the most of every journey.
          </p>

          {/* Quick search entry point */}
          <div className="w-full max-w-md mt-2">
            <button
              onClick={() => navigate("/map")}
              className="w-full flex items-center gap-3 bg-white/5 hover:bg-white/8 border border-white/10 hover:border-blue-500/40 rounded-2xl px-5 py-4 text-left transition-all group"
            >
              <Search size={18} className="text-white/30 group-hover:text-blue-400 transition-colors" />
              <span className="flex-1 text-white/30 text-sm">Where do you want to go?</span>
              <ChevronRight size={16} className="text-white/20 group-hover:text-blue-400 transition-colors" />
            </button>
          </div>

          <div className="flex items-center gap-3 flex-wrap justify-center mt-1">
            <button onClick={() => navigate("/map")}
              className="flex items-center gap-2 bg-blue-500 hover:bg-blue-400 text-white text-sm font-medium px-6 py-3 rounded-xl transition-all active:scale-[0.98] shadow-lg shadow-blue-500/20">
              <Map size={16} />
              Open map
            </button>
            <button
              className="flex items-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 text-white/70 text-sm font-medium px-6 py-3 rounded-xl transition-all">
              <Compass size={16} />
              Explore nearby
            </button>
          </div>
        </div>
      </section>

      {/* ── Feature cards ───────────────────────────────────────── */}
      <section className="px-6 pb-20 max-w-5xl mx-auto">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <FeatureCard
            icon={<Navigation size={20} />}
            color="blue"
            title="Smart routing"
            description="A→B routing across car, bus, MRT, and bicycle — adapts to live traffic and crowd conditions."
            onClick={() => navigate("/map")}
          />
          <FeatureCard
            icon={<Clock size={20} />}
            color="indigo"
            title="Proximity search"
            description='Ask "what can I reach within 20 minutes?" and explore places by category within any time window.'
            onClick={() => navigate("/map")}
          />
          <FeatureCard
            icon={<Users size={20} />}
            color="violet"
            title="Group meetup"
            description="Drop in everyone's location and find the fairest meeting point for the whole group instantly."
            onClick={() => navigate("/map")}
          />
        </div>
      </section>

      {/* ── Stats row ───────────────────────────────────────────── */}
      <section className="border-t border-white/5 px-6 py-12">
        <div className="max-w-3xl mx-auto grid grid-cols-3 gap-6 text-center">
          <Stat value="5+" label="Transport modes" />
          <Stat value="20 min" label="Avg commute saved" />
          <Stat value="Real-time" label="Crowd & traffic data" />
        </div>
      </section>

    </div>
  )
}

// ── Sub-components ────────────────────────────────────────────────────────────

const colorMap = {
  blue:   { bg: "bg-blue-500/10",   icon: "text-blue-400",   border: "hover:border-blue-500/30" },
  indigo: { bg: "bg-indigo-500/10", icon: "text-indigo-400", border: "hover:border-indigo-500/30" },
  violet: { bg: "bg-violet-500/10", icon: "text-violet-400", border: "hover:border-violet-500/30" },
}

function FeatureCard({
  icon, color, title, description, onClick,
}: {
  icon: ReactNode
  color: keyof typeof colorMap
  title: string
  description: string
  onClick: () => void
}) {
  const c = colorMap[color]
  return (
    <button onClick={onClick}
      className={`text-left bg-white/3 hover:bg-white/5 border border-white/8 ${c.border} rounded-2xl p-6 transition-all group`}>
      <div className={`w-10 h-10 ${c.bg} rounded-xl flex items-center justify-center ${c.icon} mb-4`}>
        {icon}
      </div>
      <h3 className="font-medium text-white mb-2">{title}</h3>
      <p className="text-sm text-white/40 leading-relaxed">{description}</p>
      <div className={`flex items-center gap-1 mt-4 text-xs ${c.icon} opacity-0 group-hover:opacity-100 transition-opacity`}>
        <span>Try it</span>
        <ChevronRight size={12} />
      </div>
    </button>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-2xl font-semibold text-white">{value}</span>
      <span className="text-sm text-white/40">{label}</span>
    </div>
  )
}