import { create } from 'zustand'
import {
  driverAPI,
  financialAPI,
  type Driver,
  type DispatchOffer,
  type ActiveOrder,
} from '@/lib/api'
import { useAuth } from './auth'

interface DriverState {
  driver: Driver | null
  offers: DispatchOffer[]
  activeOrder: ActiveOrder | null
  orderHistory: ActiveOrder[]
  wallet: any | null
  transactions: any[]
  isLoading: boolean
  error: string | null

  fetchDriver: () => Promise<void>
  setOnline: () => Promise<void>
  setOffline: () => Promise<void>
  updateLocation: (lat: number, lng: number, bearing?: number, speed?: number, accuracy?: number) => Promise<void>
  refreshOffers: () => Promise<void>
  refreshActiveOrder: () => Promise<void>
  refreshHistory: () => Promise<void>
  refreshWallet: () => Promise<void>
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
  orderHistory: [],
  wallet: null,
  transactions: [],
  isLoading: false,
  error: null,

  fetchDriver: async () => {
    const auth = useAuth.getState()
    if (!auth.userID) {
      set({ error: 'لم يتم العثور على معرف المندوب', driver: null })
      return
    }
    try {
      const driver = await driverAPI.getDriverByUserID(auth.userID)
      set({ driver, error: null })
    } catch (err: any) {
      set({ error: err.message, driver: null })
    }
  },

  setOnline: async () => {
    const { driver } = get()
    if (!driver) throw new Error('بيانات المندوب غير متاحة')
    const updated = await driverAPI.goOnline(driver.id)
    set({ driver: updated, error: null })
  },

  setOffline: async () => {
    const { driver } = get()
    if (!driver) throw new Error('بيانات المندوب غير متاحة')
    const updated = await driverAPI.goOffline(driver.id)
    set({ driver: updated, error: null })
  },

  updateLocation: async (lat, lng, bearing = 0, speed = 0, accuracy = 0) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.updateLocation(driver.id, {
        lat, lng, bearing, speed, accuracy,
        captured_at: new Date().toISOString(),
      })
    } catch {
      // Silent fail — location updates are best-effort
    }
  },

  refreshOffers: async () => {
    const { driver } = get()
    if (!driver) return
    try {
      const result = await driverAPI.listOffersByDriver(driver.id, 10, 0)
      const pending = (result.items || []).filter(o => o.status === 'pending')
      set({ offers: pending, error: null })
    } catch {
      // Silent fail
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
    } catch {
      set({ activeOrder: null })
    }
  },

  refreshHistory: async () => {
    try {
      const result = await driverAPI.listMyDriverOrders(50, 0)
      set({ orderHistory: result.items || [], error: null })
    } catch {
      set({ orderHistory: [] })
    }
  },

  refreshWallet: async () => {
    const { driver } = get()
    if (!driver) return
    try {
      const wallet = await financialAPI.getWalletByOwner('driver', driver.id)
      set({ wallet, error: null })
      if (wallet?.id) {
        const txResult = await financialAPI.listTransactions(wallet.id, 50, 0)
        set({ transactions: txResult.items || [] })
      }
    } catch {
      set({ wallet: null, transactions: [] })
    }
  },

  acceptOffer: async (offerId) => {
    const { driver } = get()
    if (!driver) return
    try {
      await driverAPI.acceptOffer(offerId, driver.id)
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
    try {
      await driverAPI.markPickedUp(orderId)
      await get().refreshActiveOrder()
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  markDelivered: async (orderId) => {
    try {
      await driverAPI.markDelivered(orderId)
      await get().fetchDriver()
      set({ activeOrder: null })
    } catch (err: any) {
      set({ error: err.message })
      throw err
    }
  },

  clear: () => {
    set({ driver: null, offers: [], activeOrder: null, orderHistory: [], wallet: null, transactions: [], error: null })
  },
}))
