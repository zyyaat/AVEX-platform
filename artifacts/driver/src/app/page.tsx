import { useState, useEffect, useRef } from 'react'
import { useRouter } from '@/lib/navigation'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Headphones, Menu, Power, Navigation, Clock,
  Loader2, ChevronDown, Phone, MessageCircle, X,
  Store, User, Package,
} from 'lucide-react'
import mapboxgl from 'mapbox-gl'
import 'mapbox-gl/dist/mapbox-gl.css'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { useWebSocket } from '@/hooks/use-websocket'
import { useLocationTracking } from '@/hooks/use-location-tracking'
import { OfferModal } from '@/components/OfferModal'
import { ActiveDelivery } from '@/components/ActiveDelivery'
import { SideDrawer } from '@/components/SideDrawer'
import { toast } from 'sonner'

const MAPBOX_TOKEN = import.meta.env.VITE_MAPBOX_TOKEN || 'MAPBOX_PUBLIC_TOKEN_PLACEHOLDER'

export default function DriverHome() {
  const router = useRouter()
  const { isAuthenticated, logout, userID } = useAuth()
  const {
    driver, offers, activeOrder,
    fetchDriver, setOnline, setOffline, clear,
  } = useDriver()

  const [bootChecked, setBootChecked] = useState(false)
  const [togglingOnline, setTogglingOnline] = useState(false)
  const [activeOffer, setActiveOffer] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [bottomCardExpanded, setBottomCardExpanded] = useState(false)

  const mapContainerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<mapboxgl.Map | null>(null)
  const driverMarkerRef = useRef<mapboxgl.Marker | null>(null)

  // ===== WebSocket =====
  const { isConnected, subscribe } = useWebSocket({
    onMessage: (msg) => {
      // Handle incoming WebSocket messages
      switch (msg.type) {
        case 'dispatch.offer_created':
          // New offer — refresh offers
          useDriver.getState().refreshOffers()
          toast.info('عرض جديد متاح!')
          // Haptic feedback
          if (navigator.vibrate) navigator.vibrate(200)
          break
        case 'order.status_changed':
          // Order status changed — refresh active order
          useDriver.getState().refreshActiveOrder()
          break
        case 'driver.status_changed':
          // Driver status changed — refresh driver
          useDriver.getState().fetchDriver()
          break
      }
    },
  })

  // ===== Location tracking =====
  useLocationTracking({
    enabled: driver?.status === 'online' || driver?.status === 'busy',
    interval: 5000,
  })

  // ===== Boot =====
  useEffect(() => {
    if (!isAuthenticated) {
      router.replace('/login')
      return
    }
    setBootChecked(true)
    fetchDriver()
  }, [isAuthenticated, router, fetchDriver])

  // ===== Subscribe to WebSocket channels =====
  useEffect(() => {
    if (!isConnected || !driver) return
    // Subscribe to the driver's own channel
    subscribe(`driver:${driver.id}`)
    // If driver has an active order, subscribe to that too
    if (driver.current_order_id) {
      subscribe(`order:${driver.current_order_id}`)
    }
  }, [isConnected, driver, subscribe])

  // ===== Initialize Mapbox =====
  useEffect(() => {
    if (!mapContainerRef.current || mapRef.current) return

    mapboxgl.accessToken = MAPBOX_TOKEN
    const map = new mapboxgl.Map({
      container: mapContainerRef.current,
      style: 'mapbox://styles/mapbox/streets-v12',
      center: [31.2357, 30.0444], // Cairo
      zoom: 13,
      attributionControl: false,
    })

    map.addControl(new mapboxgl.NavigationControl(), 'top-left')

    map.on('load', () => {
      // Try to get user's current location
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (pos) => {
            map.setCenter([pos.coords.longitude, pos.coords.latitude])
          },
          () => {},
          { enableHighAccuracy: true, timeout: 5000 }
        )
      }
    })

    mapRef.current = map

    return () => {
      map.remove()
      mapRef.current = null
    }
  }, [])

  // ===== Update driver marker on map =====
  useEffect(() => {
    if (!mapRef.current || !driver) return
    // Try to get driver location from the map or API
    // For now, center on Cairo if no location
  }, [driver])

  // ===== Auto-refresh offers =====
  useEffect(() => {
    if (!driver || driver.status !== 'online') return
    const interval = setInterval(() => {
      useDriver.getState().refreshOffers()
    }, 10000) // poll every 10s as fallback for WebSocket
    return () => clearInterval(interval)
  }, [driver])

  // ===== Handle new offer =====
  useEffect(() => {
    if (offers.length > 0 && !activeOffer && !activeOrder) {
      setActiveOffer(offers[0].id)
    }
  }, [offers, activeOffer, activeOrder])

  // ===== Toggle online/offline =====
  const handleToggleOnline = async () => {
    setTogglingOnline(true)
    try {
      if (driver?.status === 'online') {
        await setOffline()
        toast.success('أنت الآن غير متصل')
      } else {
        await setOnline()
        toast.success('أنت الآن متصل — بانتظار الطلبات')
      }
    } catch (err: any) {
      toast.error(err.message || 'فشل تغيير الحالة')
    } finally {
      setTogglingOnline(false)
    }
  }

  if (!bootChecked) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-white">
        <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
      </div>
    )
  }

  const isOnline = driver?.status === 'online' || driver?.status === 'busy'

  return (
    <div className="min-h-dvh bg-white relative overflow-hidden" dir="rtl">
      {/* ===== Full-screen Map ===== */}
      <div ref={mapContainerRef} className="absolute inset-0" />

      {/* ===== Top Bar ===== */}
      <div
        className="absolute top-0 left-0 right-0 z-10 flex items-center justify-between px-4 h-14"
        style={{
          backgroundColor: 'rgba(91, 192, 222, 0.95)',
          backdropFilter: 'blur(8px)',
          paddingTop: 'env(safe-area-inset-top, 0px)',
        }}
      >
        {/* Support */}
        <button
          onClick={() => router.push('/support')}
          className="w-9 h-9 rounded-full bg-white/20 flex items-center justify-center text-white hover:bg-white/30 transition-colors"
        >
          <Headphones className="w-5 h-5" />
        </button>

        {/* Driver name + status */}
        <div className="flex items-center gap-2">
          <span className="text-white font-medium text-sm">المندوب</span>
          <div className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-green-400' : 'bg-gray-400'} animate-pulse`} />
        </div>

        {/* Menu */}
        <button
          onClick={() => setDrawerOpen(true)}
          className="w-9 h-9 rounded-full bg-white/20 flex items-center justify-center text-white hover:bg-white/30 transition-colors"
        >
          <Menu className="w-5 h-5" />
        </button>
      </div>

      {/* ===== Recenter button ===== */}
      <button
        onClick={() => {
          if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition((pos) => {
              mapRef.current?.setCenter([pos.coords.longitude, pos.coords.latitude])
            })
          }
        }}
        className="absolute bottom-32 left-4 z-10 w-10 h-10 rounded-full bg-white shadow-lg flex items-center justify-center hover:bg-gray-50 transition-colors"
      >
        <Navigation className="w-5 h-5 text-gray-700" />
      </button>

      {/* ===== Online/Offline toggle ===== */}
      <button
        onClick={handleToggleOnline}
        disabled={togglingOnline}
        className="absolute bottom-32 right-4 z-10 flex items-center gap-2 px-4 h-10 rounded-full shadow-lg font-medium text-sm transition-all active:scale-95 disabled:opacity-50"
        style={{
          backgroundColor: isOnline ? '#FF6B35' : '#fff',
          color: isOnline ? '#fff' : '#333',
        }}
      >
        {togglingOnline ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : (
          <Power className="w-4 h-4" />
        )}
        {isOnline ? 'متصل' : 'غير متصل'}
      </button>

      {/* ===== Bottom Card ===== */}
      <div className="absolute bottom-0 left-0 right-0 z-10">
        {activeOrder ? (
          <ActiveDelivery />
        ) : (
          <motion.div
            initial={{ y: 100 }}
            animate={{ y: 0 }}
            className="bg-white rounded-t-2xl shadow-2xl px-5 py-4 pb-6"
            style={{ paddingBottom: 'calc(1.5rem + env(safe-area-inset-bottom, 0px))' }}
          >
            <div className="flex items-center justify-between mb-2">
              <p className="text-gray-800 text-sm font-medium">
                {isOnline ? 'لا يوجد طلبات حالياً' : 'أنت غير متصل'}
              </p>
              <button
                onClick={() => setBottomCardExpanded(!bottomCardExpanded)}
                className="text-gray-400"
              >
                <ChevronDown className={`w-5 h-5 transition-transform ${bottomCardExpanded ? 'rotate-180' : ''}`} />
              </button>
            </div>
            <p className="text-gray-500 text-xs">
              {isOnline
                ? 'يمكنك الانتظار للحصول على طلب جديد'
                : 'اضغط على زر "متصل" للبدء في استقبال الطلبات'}
            </p>

            {bottomCardExpanded && (
              <div className="mt-3 pt-3 border-t border-gray-100 space-y-2">
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Clock className="w-4 h-4" />
                  <span>الشيفت الحالي: {new Date().toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })}</span>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Package className="w-4 h-4" />
                  <span>طلبات اليوم: {driver?.total_deliveries || 0}</span>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <User className="w-4 h-4" />
                  <span>التقييم: {driver?.rating.toFixed(1) || '5.0'} ⭐</span>
                </div>
              </div>
            )}
          </motion.div>
        )}
      </div>

      {/* ===== Offer Modal ===== */}
      <AnimatePresence>
        {activeOffer && (
          <OfferModal
            offer={offers.find((o) => o.id === activeOffer)!}
            onClose={() => setActiveOffer(null)}
          />
        )}
      </AnimatePresence>

      {/* ===== Side Drawer ===== */}
      <SideDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />

      {/* ===== Connection indicator ===== */}
      {!isConnected && isOnline && (
        <div className="absolute top-16 left-1/2 -translate-x-1/2 z-20 bg-yellow-100 text-yellow-800 text-xs px-3 py-1 rounded-full shadow">
          جاري إعادة الاتصال...
        </div>
      )}
    </div>
  )
}
