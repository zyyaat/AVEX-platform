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
          // The Go backend returns { token, driver: { id, ... } } for driver logins.
          const userID = result.driver?.id || ''
          if (!userID) {
            console.error('Login: no driver.id in response', result)
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
          // Validate the token by fetching the driver profile.
          // If it's expired or invalid, the backend returns 401 and we log out.
          try {
            const { driverAPI } = await import('@/lib/api')
            // Try to fetch the driver profile using the stored userID.
            const userID = get().userID
            if (userID) {
              await driverAPI.getDriverByUserID(userID)
              // Token is valid — keep the user logged in.
              set({ isAuthenticated: true })
            }
          } catch (err: any) {
            // Token is invalid or expired — log out gracefully.
            console.warn('Token validation failed, logging out:', err.message)
            setAuthToken(null)
            set({
              token: null,
              userID: null,
              role: null,
              isAuthenticated: false,
            })
          }
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
