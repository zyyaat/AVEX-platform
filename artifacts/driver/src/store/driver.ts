import { create } from 'zustand'
import {
  driverAPI,
  type Driver,
  type DispatchOffer,
  type ActiveOrder,
} from '@/lib/api'
import { useAuth } from './auth'

interface DriverState {
  driver: Driver | null
  offers: DispatchOffer[]
  activeOrder: ActiveOrder | null
  isLoading: boolean
  error: string | null

  fetchDriver: () => Promise<void>
  setOnline: () => Promise<void>
  setOffline: () => Promise<void>
  updateLocation: (lat: number, lng: number, bearing?: number, speed?: number, accuracy?: number) => Promise<void>
  refreshOffers: () => Promise<void>
  refreshActiveOrder: () => Promise<void>
  acceptOffer: (offerId: string) => Promise<void>
  rejectOffer: (offerId: string, reason?: string) => Promise<void>
  markPickedUp: (orderId: string) => Promise<void>
  markDelivered: (orderId: string) => Promise<void>
  clear: () => void
}

export const useDriver = create<DriverState>((set, get) => ({
  driver: null,
  offers: [],
  activeOrder: null,
  isLoading: false,
  error: null,

  fetchDriver: async () => {
    const auth = useAuth.getState()
    if (!auth.userID) return
    try {
      const driver = await driverAPI.getDriverByUserID(auth.userID)
      set({ driver, error: null })
    } catch (err: any) {
      console.error('fetchDriver error:', err.message)
      set({ error: err.message, driver: null })
    }
  },

  setOnline: async () => {
    const { driver } = get()
    if (!driver) return
    try {
      const updated = await driverAPI.goOnline(driver.id)
      set({ driver: updated, error: null })
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  setOffline: async () => {
    const { driver } = get()
    if (!driver) return
    try {
      const updated = await driverAPI.goOffline(driver.id)
      set({ driver: updated, error: null })
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  updateLocation: async (lat, lng, bearing = 0, speed = 0, accuracy = 0) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.updateLocation(driver.id, {
        lat, lng, bearing, speed, accuracy,
        captured_at: new Date().toISOString(),
      })
    } catch (err: any) {
      // Silent fail — location updates are best-effort
      console.error('Location update failed:', err.message)
    }
  },

  refreshOffers: async () => {
    const { driver } = get()
    if (!driver) return
    try {
      const result = await driverAPI.listOffersByDriver(driver.id, 10, 0)
      // Filter only pending offers
      const pending = (result.items || []).filter(o => o.status === 'pending')
      set({ offers: pending, error: null })
    } catch (err: any) {
      set({ error: err.message })
    }
  },

  refreshActiveOrder: async () => {
    const { driver } = get()
    if (!driver || !driver.current_order_id) {
      set({ activeOrder: null })
      return
    }
    try {
      const order = await driverAPI.getOrder(driver.current_order_id)
      set({ activeOrder: order, error: null })
    } catch (err: any) {
      set({ error: err.message })
    }
  },

  acceptOffer: async (offerId) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.acceptOffer(offerId, driver.id)
      // Refresh driver state + active order
      await get().fetchDriver()
      await get().refreshActiveOrder()
      set((state) => ({ offers: state.offers.filter(o => o.id !== offerId) }))
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  rejectOffer: async (offerId, reason) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.rejectOffer(offerId, driver.id, reason)
      set((state) => ({ offers: state.offers.filter(o => o.id !== offerId) }))
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  markPickedUp: async (orderId) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.markPickedUp(orderId, driver.id)
      await get().refreshActiveOrder()
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  markDelivered: async (orderId) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.markDelivered(orderId, driver.id)
      await get().fetchDriver()
      set({ activeOrder: null })
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  clear: () => {
    set({ driver: null, offers: [], activeOrder: null, error: null })
  },
}))
