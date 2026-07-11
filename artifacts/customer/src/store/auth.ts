
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { authAPI, setAuthToken, type User } from '@/lib/api'

interface AuthState {
  user: User | null
  token: string | null
  isLoading: boolean
  isAuthenticated: boolean
  isInitialized: boolean  // ← tracks whether initialize() has run

  login: (phone: string, password: string) => Promise<void>
  register: (name: string, phone: string, password: string, email?: string) => Promise<void>
  logout: () => void
  fetchUser: () => Promise<void>
  initialize: () => Promise<void>
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isLoading: false,
      isAuthenticated: false,
      isInitialized: false,

      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const { token, user } = await authAPI.login({ phone, password })
          setAuthToken(token)
          set({ user, token, isAuthenticated: true, isLoading: false, isInitialized: true })
        } catch (err) {
          set({ isLoading: false })
          throw err
        }
      },

      register: async (name, phone, password, email) => {
        set({ isLoading: true })
        try {
          const { token, user } = await authAPI.register({ name, phone, password, email })
          setAuthToken(token)
          set({ user, token, isAuthenticated: true, isLoading: false, isInitialized: true })
        } catch (err) {
          set({ isLoading: false })
          throw err
        }
      },

      logout: () => {
        setAuthToken(null)
        set({ user: null, token: null, isAuthenticated: false })
      },

      fetchUser: async () => {
        const token = get().token
        if (!token) return
        try {
          const user = await authAPI.me()
          set({ user, isAuthenticated: true })
        } catch {
          setAuthToken(null)
          set({ user: null, token: null, isAuthenticated: false })
        }
      },

      initialize: async () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
          await get().fetchUser()
        }
        // Mark as initialized whether or not we had a token.
        set({ isInitialized: true })
      },
    }),
    {
      name: 'avex-auth',
      partialize: (state) => ({ token: state.token, user: state.user }),
    }
  )
)
