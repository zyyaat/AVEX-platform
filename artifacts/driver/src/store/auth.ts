import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { driverAuthAPI, setAuthToken, getAuthToken } from '@/lib/api'

interface AuthState {
  token: string | null
  userID: string | null
  role: string | null
  isLoading: boolean
  isAuthenticated: boolean

  login: (phone: string, password: string) => Promise<void>
  logout: () => void
  initialize: () => Promise<void>
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      userID: null,
      role: null,
      isLoading: false,
      isAuthenticated: false,

      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const result = await driverAuthAPI.login({ phone, password })
          setAuthToken(result.token)
          // The Go backend returns { token, user: { id, ... } }
          const userID = result.user?.id || ''
          if (!userID) {
            console.error('Login: no user.id in response', result)
            throw new Error('فشل تسجيل الدخول — استجابة غير صحيحة من الخادم')
          }
          console.log('Login success, userID:', userID)
          set({
            token: result.token,
            userID,
            role: 'driver',
            isAuthenticated: true,
            isLoading: false,
          })
        } catch (err) {
          set({ isLoading: false })
          throw err
        }
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

      initialize: async () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
        }
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
