import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { adminAuthAPI, setAuthToken } from '@/lib/api'

interface AuthState {
  token: string | null
  user: any
  isAuthenticated: boolean
  isLoading: boolean
  isInitialized: boolean
  login: (phone: string, password: string) => Promise<void>
  logout: () => void
  initialize: () => Promise<void>
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null, user: null, isAuthenticated: false, isLoading: false,
      isInitialized: false,
      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const result = await adminAuthAPI.login({ phone, password })
          let { token, user } = result
          if (!user) {
            setAuthToken(token)
            user = await adminAuthAPI.me()
          }
          const isAdmin = user?.is_admin === true || user?.isAdmin === true || user?.role === 'admin'
          if (!isAdmin) throw new Error('هذا الحساب ليس مديراً')
          setAuthToken(token)
          set({ token, user, isAuthenticated: true, isLoading: false })
        } catch (e) { set({ isLoading: false }); throw e }
      },
      logout: () => { setAuthToken(null); set({ token: null, user: null, isAuthenticated: false }) },
      initialize: async () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
          try {
            const u = await adminAuthAPI.me()
            if (u?.is_admin || u?.isAdmin) set({ user: u, isAuthenticated: true })
            else set({ token: null, isAuthenticated: false })
          } catch { setAuthToken(null); set({ token: null, isAuthenticated: false }) }
        }
      },
    }),
    { name: 'avex-admin-auth', partialize: (s) => ({ token: s.token, user: s.user, isAuthenticated: s.isAuthenticated }) }
  )
)
