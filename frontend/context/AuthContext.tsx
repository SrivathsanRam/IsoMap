import { createContext, useContext, useCallback, useEffect, useMemo, useState, type ReactNode } from "react"
import type { User } from "@/types"
import { api } from "@/services/api"

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  loginWithGoogle: (credential: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    let active = true

    api.getCurrentUser()
      .then((currentUser) => {
        if (active) {
          setUser(currentUser)
        }
      })
      .catch(() => {
        if (active) {
          setUser(null)
        }
      })
      .finally(() => {
        if (active) {
          setIsLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [])

  const loginWithGoogle = useCallback(async (credential: string) => {
    setIsLoading(true)
    try {
      const currentUser = await api.loginWithGoogle(credential)
      setUser(currentUser)
    } finally {
      setIsLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    setIsLoading(true)
    try {
      await api.logout()
    } catch {
      // ignore logout errors
    } finally {
      setUser(null)
      setIsLoading(false)
    }
  }, [])

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isAuthenticated: !!user, 
      isLoading,
      loginWithGoogle,
      logout,
    }),
    [user, isLoading, loginWithGoogle, logout]
  )

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}

// Export hook separately for Fast Refresh compatibility
// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
