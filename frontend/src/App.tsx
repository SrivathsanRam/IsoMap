import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AuthProvider, useAuth } from "../context/AuthContext"
import { LoginPage } from "../components/Auth/LoginPage"
import LandingPage from "../pages/LandingPage"
import OutingPage from "../pages/OutingPage"
import { Map } from "./components/Map"

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return null
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/outings/:token" element={<OutingPage />} />

      <Route path="/" element={
        <ProtectedRoute><LandingPage /></ProtectedRoute>
      } />

      <Route path="/map" element={
        <ProtectedRoute><Map /></ProtectedRoute>
      } />

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
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
