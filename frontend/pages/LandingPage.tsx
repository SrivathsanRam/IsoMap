import type { ReactNode } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../context/AuthContext"
import {
  ChevronRight,
  Clock,
  Compass,
  LogOut,
  Map,
  MapPin,
  Navigation,
  Search,
  Users,
  Zap,
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
      <header className="border-b border-white/10 bg-slate-950/90">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-500 shadow-lg shadow-blue-500/20">
              <MapPin size={18} className="text-white" />
            </div>
            <div>
              <div className="font-semibold tracking-tight">ISOMAP</div>
              <div className="flex items-center gap-1 text-xs text-white/45">
                <Zap size={12} />
                Journey-centred navigation
              </div>
            </div>
          </div>

          <nav className="hidden items-center rounded-lg border border-white/10 bg-white/[0.03] p-1 text-sm text-white/60 md:flex">
            <button
              onClick={() => navigate("/map")}
              className="rounded-md px-4 py-2 hover:bg-white/10 hover:text-white"
            >
              Map
            </button>
            <button className="rounded-md px-4 py-2 hover:bg-white/10 hover:text-white">
              Proximity
            </button>
            <button className="rounded-md px-4 py-2 hover:bg-white/10 hover:text-white">
              Group
            </button>
          </nav>

          <div className="flex items-center gap-3">
            <span className="hidden max-w-52 truncate text-sm text-white/55 sm:block">
              Hey, <span className="text-white/85">{user?.name}</span>
            </span>
            <button
              onClick={handleLogout}
              className="flex items-center gap-1.5 rounded-lg border border-white/10 px-3 py-2 text-sm text-white/65 hover:border-red-400/40 hover:text-red-300"
            >
              <LogOut size={15} />
              <span className="hidden sm:block">Sign out</span>
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6">
        <section className="grid gap-8 py-14 lg:grid-cols-[1.1fr_0.9fr] lg:items-center lg:py-20">
          <div className="max-w-2xl">
            <h1 className="text-4xl font-semibold leading-tight tracking-tight sm:text-5xl">
              Navigate smarter.
              <br />
              <span className="text-blue-400">Travel better.</span>
            </h1>

            <p className="mt-5 max-w-xl text-base leading-7 text-white/60 sm:text-lg">
              More than the fastest route - discover what you can reach, coordinate with friends,
              and make the most of every journey.
            </p>

            <div className="mt-8 max-w-md">
              <button
                onClick={() => navigate("/map")}
                className="group flex w-full items-center gap-3 rounded-xl border border-white/10 bg-white/[0.04] px-5 py-4 text-left transition hover:border-blue-400/50 hover:bg-white/[0.07]"
              >
                <Search size={19} className="text-white/35 group-hover:text-blue-300" />
                <span className="flex-1 text-sm text-white/55">Where do you want to go?</span>
                <ChevronRight size={17} className="text-white/30 group-hover:text-blue-300" />
              </button>
            </div>

            <div className="mt-4 flex flex-wrap gap-3">
              <button
                onClick={() => navigate("/map")}
                className="flex items-center gap-2 rounded-lg bg-blue-500 px-5 py-3 text-sm font-medium text-white shadow-lg shadow-blue-500/20 transition hover:bg-blue-400"
              >
                <Map size={16} />
                Open map
              </button>
              <button className="flex items-center gap-2 rounded-lg border border-white/10 bg-white/[0.04] px-5 py-3 text-sm font-medium text-white/70 transition hover:bg-white/[0.08] hover:text-white">
                <Compass size={16} />
                Explore nearby
              </button>
            </div>
          </div>

          <div className="grid gap-3 rounded-xl border border-white/10 bg-white/[0.03] p-4 sm:grid-cols-3 lg:grid-cols-1">
            <Stat value="5+" label="Transport modes" />
            <Stat value="20 min" label="Avg commute saved" />
            <Stat value="Real-time" label="Crowd & traffic data" />
          </div>
        </section>

        <section className="pb-16">
          <div className="grid gap-4 md:grid-cols-3">
            <FeatureCard
              icon={<Navigation size={20} />}
              color="blue"
              title="Smart routing"
              description="A to B routing across car, bus, MRT, and bicycle - adapts to live traffic and crowd conditions."
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
      </main>
    </div>
  )
}

const colorMap = {
  blue: { bg: "bg-blue-500/10", icon: "text-blue-400", border: "hover:border-blue-500/30" },
  indigo: { bg: "bg-indigo-500/10", icon: "text-indigo-400", border: "hover:border-indigo-500/30" },
  violet: { bg: "bg-violet-500/10", icon: "text-violet-400", border: "hover:border-violet-500/30" },
}

function FeatureCard({
  icon,
  color,
  title,
  description,
  onClick,
}: {
  icon: ReactNode
  color: keyof typeof colorMap
  title: string
  description: string
  onClick: () => void
}) {
  const c = colorMap[color]
  return (
    <button
      onClick={onClick}
      className={`group rounded-xl border border-white/10 bg-white/[0.03] p-6 text-left transition hover:bg-white/[0.06] ${c.border}`}
    >
      <div className={`mb-4 flex h-10 w-10 items-center justify-center rounded-lg ${c.bg} ${c.icon}`}>
        {icon}
      </div>
      <h3 className="mb-2 font-medium text-white">{title}</h3>
      <p className="text-sm leading-6 text-white/50">{description}</p>
      <div className={`mt-5 flex items-center gap-1 text-xs ${c.icon}`}>
        <span>Try it</span>
        <ChevronRight size={12} />
      </div>
    </button>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="rounded-lg px-3 py-4 text-center">
      <div className="text-xl font-semibold text-white">{value}</div>
      <div className="mt-1 text-xs text-white/45">{label}</div>
    </div>
  )
}
