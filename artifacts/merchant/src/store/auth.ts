import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { merchantAuthAPI, setAuthToken, type Merchant } from '@/lib/api'

interface AuthState {
  token: string | null
  merchant: Merchant | null
  isAuthenticated: boolean
  isLoading: boolean
  mustChangePassword: boolean

  login: (phone: string, password: string) => Promise<{ mustChangePassword: boolean }>
  logout: () => void
  fetchMe: () => Promise<void>
  setMustChangePassword: (v: boolean) => void
  initialize: () => Promise<void>
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      merchant: null,
      isAuthenticated: false,
      isLoading: false,
      mustChangePassword: false,

      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const result = await merchantAuthAPI.login({ phone, password })
    const { token } = result
    const mustChangePassword = result.must_change_password ?? false
    const merchant = result.user || result.merchant
          setAuthToken(token)
          set({
            token,
            merchant,
            mustChangePassword,
            isAuthenticated: true,
            isLoading: false,
          })
          return { mustChangePassword }
        } catch (e) {
          set({ isLoading: false })
          throw e
        }
      },

      logout: () => {
        setAuthToken(null)
        set({
          token: null,
          merchant: null,
          isAuthenticated: false,
          mustChangePassword: false,
        })
      },

      fetchMe: async () => {
        try {
          const m = await merchantAuthAPI.me()
          set({ merchant: m })
        } catch {}
      },

      setMustChangePassword: (v) => set({ mustChangePassword: v }),

      initialize: async () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
          try {
            const m = await merchantAuthAPI.me()
            set({ merchant: m, isAuthenticated: true })
          } catch {
            setAuthToken(null)
            set({ token: null, isAuthenticated: false })
          }
        }
      },
    }),
    {
      name: 'avex-merchant-auth',
      partialize: (s) => ({
        token: s.token,
        merchant: s.merchant,
        isAuthenticated: s.isAuthenticated,
        mustChangePassword: s.mustChangePassword,
      }),
    }
  )
)
