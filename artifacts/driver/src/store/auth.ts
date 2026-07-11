import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { driverAuthAPI, setAuthToken } from '@/lib/api'

interface AuthState {
  token: string | null
  userID: string | null
  role: string | null
  isAuthenticated: boolean

  login: (phone: string, password: string) => Promise<void>
  logout: () => void
  initialize: () => void
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      userID: null,
      role: null,
      isAuthenticated: false,

      login: async (phone, password) => {
        const result = await driverAuthAPI.login({ phone, password })
        setAuthToken(result.token)
        const userID = result.driver?.id || ''
        if (!userID) {
          throw new Error('فشل تسجيل الدخول — استجابة غير صحيحة من الخادم')
        }
        set({
          token: result.token,
          userID,
          role: 'driver',
          isAuthenticated: true,
        })
      },

      logout: () => {
        setAuthToken(null)
        set({
          token: null,
          userID: null,
          role: null,
          isAuthenticated: false,
        })
      },

      // Synchronous — just restore token from localStorage
      initialize: () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
        }
        // isAuthenticated is already restored by persist middleware
      },
    }),
    {
      name: 'avex-driver-auth',
      partialize: (state) => ({
        token: state.token,
        userID: state.userID,
        role: state.role,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
