import {
  createContext, useContext, useCallback,
  useEffect, useMemo, useState, type ReactNode
} from "react"
import type { User } from "@/types"
import { api } from "@/services/api"

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  isGuest: boolean         
  isLoading: boolean
  loginWithGoogle: (credential: string) => Promise<void>
  continueAsGuest: () => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

// Sentinel object so ProtectedRoute can tell guest from real user
const GUEST_USER: User = { id: "guest", name: "Guest" } as unknown as User

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isGuest, setIsGuest] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  
  useEffect(() => {
    if (localStorage.getItem("isGuest") === "true") {
      setUser(GUEST_USER)
      setIsGuest(true)
      setIsLoading(false)
      return
    }

    let active = true
    api.getCurrentUser()
      .then((currentUser) => { if (active) setUser(currentUser) })
      .catch(() => { if (active) setUser(null) })
      .finally(() => { if (active) setIsLoading(false) })

    return () => { active = false }
  }, [])

  const loginWithGoogle = useCallback(async (credential: string) => {
    setIsLoading(true)
    try {
      const currentUser = await api.loginWithGoogle(credential)
      setUser(currentUser)
      setIsGuest(false)
      localStorage.removeItem("isGuest")
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Guest login — no backend call, just set local state
  const continueAsGuest = useCallback(() => {
    setUser(GUEST_USER)
    setIsGuest(true)
    localStorage.setItem("isGuest", "true")
  }, [])

  const logout = useCallback(async () => {
    setIsLoading(true)
    try {
      if (!isGuest) await api.logout()
    } catch {
      // ignore
    } finally {
      setUser(null)
      setIsGuest(false)
      localStorage.removeItem("isGuest")
      setIsLoading(false)
    }
  }, [isGuest])

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isAuthenticated: !!user,
      isGuest,
      isLoading,
      loginWithGoogle,
      continueAsGuest,
      logout,
    }),
    [user, isGuest, isLoading, loginWithGoogle, continueAsGuest, logout]
  )

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}