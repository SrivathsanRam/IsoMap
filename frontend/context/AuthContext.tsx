import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from "react"
import type { User } from "@/types"
import { api } from "@/services/api"

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (username: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

function getInitialUser(): User | null {
  try {
    const storedUser = localStorage.getItem("user") 
    if (storedUser) {
      const parsedUser = JSON.parse(storedUser) as User
      api.setUserId(parsedUser.id)
      return parsedUser
    }
  } catch {
    localStorage.removeItem("user")
  }
  return null
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(getInitialUser)
  const [isLoading, setIsLoading] = useState(false)

  const login = useCallback(async (username: string) => {
    setIsLoading(true)
    try {
      await api.login(username)
      const tempUser: User = { id: "0", name: username }
      setUser(tempUser)
      localStorage.setItem("user", JSON.stringify(tempUser))
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
      localStorage.removeItem("user")
      setIsLoading(false)
    }
  }, [])

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isAuthenticated: !!user, 
      isLoading,
      login,
      logout,
    }),
    [user, isLoading, login, logout]
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
