import type { ReactNode } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../context/AuthContext"
import { createOuting } from "../src/lib/api"
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

  const handleCreateOuting = async () => {
    const outing = await createOuting("Group outing")
    const path = `/outings/${outing.join_token}`
    await navigator.clipboard.writeText(`${window.location.origin}${path}`)
    navigate(path)
  }

  return (
    <div className="min-h-screen bg-white text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900">
              <MapPin size={18} className="text-white" />
            </div>
            <div>
              <div className="font-semibold tracking-tight">ISOMAP</div>
              <div className="flex items-center gap-1 text-xs text-slate-500">
                <Zap size={12} />
                Journey-centred navigation
              </div>
            </div>
          </div>

          <nav className="hidden items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 p-1 text-sm text-slate-600 md:flex">
            <button
              onClick={() => navigate("/map")}
              className="rounded-md px-4 py-2 hover:bg-white hover:text-slate-950"
            >
              Map
            </button>
            <button className="rounded-md px-4 py-2 hover:bg-white hover:text-slate-950">
              Proximity
            </button>
            <button
              onClick={handleCreateOuting}
              className="rounded-md px-4 py-2 hover:bg-white hover:text-slate-950"
            >
              Group
            </button>
          </nav>

          <div className="flex items-center gap-3">
            <span className="hidden max-w-52 truncate text-sm text-slate-500 sm:block">
              Hey, <span className="text-slate-800">{user?.name}</span>
            </span>
            <button
              onClick={handleLogout}
              className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-600 hover:border-red-200 hover:text-red-600"
            >
              <LogOut size={15} />
              <span className="hidden sm:block">Sign out</span>
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6">
        <section className="grid gap-8 py-12 lg:grid-cols-[1.15fr_0.85fr] lg:items-center lg:py-16">
          <div className="max-w-2xl">
            <h1 className="text-4xl font-semibold leading-tight tracking-tight text-slate-950 sm:text-5xl">
              Navigate smarter.
              <br />
              <span className="text-blue-600">Travel better.</span>
            </h1>

            <p className="mt-5 max-w-xl text-base leading-7 text-slate-600 sm:text-lg">
              More than the fastest route - discover what you can reach, coordinate with friends,
              and make the most of every journey.
            </p>

            <div className="mt-8 max-w-md">
              <button
                onClick={() => navigate("/map")}
                className="group flex w-full items-center gap-3 rounded-xl border border-slate-200 bg-white px-5 py-4 text-left shadow-sm transition hover:border-blue-300 hover:shadow-md"
              >
                <Search size={19} className="text-slate-400 group-hover:text-blue-600" />
                <span className="flex-1 text-sm text-slate-500">Where do you want to go?</span>
                <ChevronRight size={17} className="text-slate-400 group-hover:text-blue-600" />
              </button>
            </div>

            <div className="mt-4 flex flex-wrap gap-3">
              <button
                onClick={() => navigate("/map")}
                className="flex items-center gap-2 rounded-lg bg-slate-900 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-700"
              >
                <Map size={16} />
                Open map
              </button>
              <button className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-5 py-3 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50">
                <Compass size={16} />
                Explore nearby
              </button>
            </div>
          </div>

          <div className="grid gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 sm:grid-cols-3 lg:grid-cols-1">
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
              onClick={handleCreateOuting}
            />
          </div>
        </section>
      </main>
    </div>
  )
}

const colorMap = {
  blue: { bg: "bg-blue-50", icon: "text-blue-600", border: "hover:border-blue-200" },
  indigo: { bg: "bg-indigo-50", icon: "text-indigo-600", border: "hover:border-indigo-200" },
  violet: { bg: "bg-violet-50", icon: "text-violet-600", border: "hover:border-violet-200" },
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
      className={`group rounded-xl border border-slate-200 bg-white p-6 text-left shadow-sm transition hover:shadow-md ${c.border}`}
    >
      <div className={`mb-4 flex h-10 w-10 items-center justify-center rounded-lg ${c.bg} ${c.icon}`}>
        {icon}
      </div>
      <h3 className="mb-2 font-medium text-slate-950">{title}</h3>
      <p className="text-sm leading-6 text-slate-600">{description}</p>
      <div className={`mt-5 flex items-center gap-1 text-xs ${c.icon}`}>
        <span>Try it</span>
        <ChevronRight size={12} />
      </div>
    </button>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="rounded-lg bg-white px-3 py-4 text-center shadow-sm">
      <div className="text-xl font-semibold text-slate-950">{value}</div>
      <div className="mt-1 text-xs text-slate-500">{label}</div>
    </div>
  )
}
