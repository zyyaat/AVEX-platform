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
        // SIMPLE & FAST: just restore the token from localStorage.
        // NO API calls during initialization — this prevents the page
        // from hanging if the backend is slow or the driver doesn't
        // exist in dispatch.drivers yet.
        //
        // If the token is expired/invalid, the first real API call
        // (fetchDriver) will return 401 and the 401 handler will log
        // the user out gracefully.
        const token = get().token
        if (token) {
          setAuthToken(token)
          // Keep isAuthenticated as-is (it was persisted as true).
          // The route guard will let the user through immediately.
          set({ isInitialized: true })
        } else {
          // No token — not authenticated.
          set({ isAuthenticated: false, isInitialized: true })
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
