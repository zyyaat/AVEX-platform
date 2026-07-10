import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { agentAuthAPI, setAuthToken } from '@/lib/api'

interface AuthState {
  token: string | null
  agent: any
  isAuthenticated: boolean
  isLoading: boolean
  login: (phone: string, password: string) => Promise<void>
  logout: () => void
  initialize: () => Promise<void>
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null, agent: null, isAuthenticated: false, isLoading: false,
      login: async (phone, password) => {
        set({ isLoading: true })
        try {
          const result = await agentAuthAPI.login({ phone, password })
    const { token } = result
    const agent = result.user || result.agent
          setAuthToken(token)
          set({ token, agent, isAuthenticated: true, isLoading: false })
        } catch (e) { set({ isLoading: false }); throw e }
      },
      logout: () => { setAuthToken(null); set({ token: null, agent: null, isAuthenticated: false }) },
      initialize: async () => {
        const token = get().token
        if (token) {
          setAuthToken(token)
          try { const a = await agentAuthAPI.me(); set({ agent: a, isAuthenticated: true }) }
          catch { setAuthToken(null); set({ token: null, isAuthenticated: false }) }
        }
      },
    }),
    { name: 'avex-agent-auth', partialize: (s) => ({ token: s.token, agent: s.agent, isAuthenticated: s.isAuthenticated }) }
  )
)
