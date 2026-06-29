import { lazy, Suspense } from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AuthProvider, useAuth } from "../context/AuthContext"
import { LoginPage } from "../components/Auth/LoginPage"
import LandingPage from "../pages/LandingPage"

const MapPage = lazy(() => import("./components/Map").then((module) => ({ default: module.Map })))
const OutingPage = lazy(() => import("../pages/OutingPage"))

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return null
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function AppRoutes() {
  return (
    <Suspense fallback={null}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/outings/:token" element={<OutingPage />} />

        <Route path="/" element={
          <ProtectedRoute><LandingPage /></ProtectedRoute>
        } />

        <Route path="/map" element={
          <ProtectedRoute><MapPage /></ProtectedRoute>
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
