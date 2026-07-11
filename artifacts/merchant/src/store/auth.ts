import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { merchantAuthAPI, setAuthToken, type Merchant } from '@/lib/api'

interface AuthState {
  token: string | null
  merchant: Merchant | null
  isAuthenticated: boolean
  isLoading: boolean
  isInitialized: boolean
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
      isInitialized: false,
      isLoading: false,
      mustChangePassword: false,

      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const result = await merchantAuthAPI.login({ phone, password })
          // After toCamelCase transform, fields are camelCase.
          // Backend returns { token, user, must_change_password }.
          // We store the user object as 'merchant' (backend doesn't have
          // a separate merchant login endpoint yet — user with merchant
          // role is used).
          const { token } = result
          const mustChangePassword = (result as any).mustChangePassword ?? false
          const merchant = result.user || (result as any).merchant
          setAuthToken(token)
          set({
            token,
            merchant,
            mustChangePassword,
            isAuthenticated: true,
            isLoading: false,
            isInitialized: true,
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
        set({ isInitialized: true })
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
