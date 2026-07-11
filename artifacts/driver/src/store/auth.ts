import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { driverAuthAPI, setAuthToken } from '@/lib/api'

interface AuthState {
  token: string | null
  userID: string | null
  role: string | null
  isLoading: boolean
  isAuthenticated: boolean
  isInitialized: boolean  // ← NEW: tracks whether initialize() has run

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
      isInitialized: false,  // ← starts false, becomes true after initialize()

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
            isInitialized: true,  // ← login counts as initialization
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
          // ONLY log out if the token is actually invalid/expired (401).
          // Other errors (network, 404, 500) should NOT log the user out —
          // they might be temporary, and the user should stay logged in.
          try {
            const { driverAPI } = await import('@/lib/api')
            const userID = get().userID
            if (userID) {
              await driverAPI.getDriverByUserID(userID)
            }
            // Success — token is valid.
            set({ isAuthenticated: true, isInitialized: true })
          } catch (err: any) {
            const errMsg = err.message || ''
            // Only log out on auth errors (401) or session-expired messages.
            // Network errors, 404s, 500s, etc. should keep the user logged in.
            if (errMsg.includes('انتهت الجلسة') || errMsg.includes('401') || errMsg.includes('invalid or expired token')) {
              console.warn('Token is invalid/expired, logging out:', errMsg)
              setAuthToken(null)
              set({
                token: null,
                userID: null,
                role: null,
                isAuthenticated: false,
                isInitialized: true,
              })
            } else {
              // Non-auth error — keep the user logged in, just mark as initialized.
              // The home page will retry the API call and show an error if needed.
              console.warn('Token validation failed (non-auth error), keeping session:', errMsg)
              set({ isAuthenticated: true, isInitialized: true })
            }
          }
        } else {
          // No token — mark as initialized so the route guard can proceed.
          set({ isInitialized: true })
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
        // NOTE: isInitialized is NOT persisted — it must be re-evaluated
        // on every page load by calling initialize().
      }),
    }
  )
)
