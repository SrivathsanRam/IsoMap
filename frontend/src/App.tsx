import { lazy, Suspense } from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AuthProvider, useAuth } from "../context/AuthContext"
import { LoginPage } from "../components/Auth/LoginPage"
import LandingPage from "../pages/LandingPage"

const MapPage = lazy(() => import("./components/Map").then((m) => ({ default: m.Map })))
const OutingPage = lazy(() => import("../pages/OutingPage"))

// Requires a real Google account — blocks guests
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isGuest, isLoading } = useAuth()
  if (isLoading) return null
  if (!isAuthenticated || isGuest) return <Navigate to="/login" replace />
  return <>{children}</>
}

// Allows both real users AND guests — blocks only logged-out users
function AuthOrGuestRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return null
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function AppRoutes() {
  return (
    <Suspense fallback={null}>
      <Routes>
        {/* Public */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/outings/:token" element={<OutingPage />} />

        {/* Guests + signed-in: landing page and map */}
        <Route path="/" element={
          <AuthOrGuestRoute><LandingPage /></AuthOrGuestRoute>
        } />
        <Route path="/map" element={
          <AuthOrGuestRoute><MapPage /></AuthOrGuestRoute>
        } />

        {/* Signed-in only: group planning */}
        <Route path="/group" element={
          <ProtectedRoute>
            <div className="p-8 text-white">Group Planning — coming soon</div>
          </ProtectedRoute>
        } />

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  )
}