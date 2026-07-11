import { useState, useEffect, useRef } from 'react'
import { useRouter } from '@/lib/navigation'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Headphones, Menu, Power, Navigation, Clock,
  Loader2, ChevronDown, Phone, MessageCircle, X,
  Store, User, Package,
} from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { useWebSocket } from '@/hooks/use-websocket'
import { useLocationTracking } from '@/hooks/use-location-tracking'
import { OfferModal } from '@/components/OfferModal'
import { ActiveDelivery } from '@/components/ActiveDelivery'
import { SideDrawer } from '@/components/SideDrawer'
import { toast } from 'sonner'

// ===== Mapbox CDN Loader =====
// Read the Mapbox token from Vite env (VITE_MAPBOX_TOKEN) or fall back to
// the global MAPBOX_ACCESS_TOKEN (set by Replit Secrets via window.__ENV__).
// The token is a PUBLIC token (safe to expose in browser code).
const MAPBOX_TOKEN: string =
  (import.meta.env.VITE_MAPBOX_TOKEN as string) ||
  (typeof window !== 'undefined' && (window as any).__MAPBOX_ACCESS_TOKEN) ||
  ''

function loadMapbox(): Promise<any> {
  // Load CSS
  if (!document.querySelector('#mapbox-gl-css')) {
    const link = document.createElement('link')
    link.id = 'mapbox-gl-css'
    link.rel = 'stylesheet'
    link.href = 'https://api.mapbox.com/mapbox-gl-js/v3.5.2/mapbox-gl.css'
    document.head.appendChild(link)
  }
  // Load JS
  if ((window as any).mapboxgl) return Promise.resolve((window as any).mapboxgl)
  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = 'https://api.mapbox.com/mapbox-gl-js/v3.5.2/mapbox-gl.js'
    script.onload = () => resolve((window as any).mapboxgl)
    script.onerror = () => reject(new Error('Failed to load Mapbox'))
    document.head.appendChild(script)
  })
}

export default function DriverHome() {
  const router = useRouter()
  const { isAuthenticated, isInitialized, userID } = useAuth()
  const {
    driver, offers, activeOrder, error,
    fetchDriver, setOnline, setOffline,
  } = useDriver()

  const [bootChecked, setBootChecked] = useState(false)
  const [togglingOnline, setTogglingOnline] = useState(false)
  const [activeOffer, setActiveOffer] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [bottomCardExpanded, setBottomCardExpanded] = useState(false)
  const [mapReady, setMapReady] = useState(false)
  const [mapError, setMapError] = useState<string | null>(null)

  const mapContainerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<any>(null)

  // ===== WebSocket =====
  const { isConnected, subscribe } = useWebSocket({
    onMessage: (msg) => {
      switch (msg.type) {
        case 'dispatch.offer_created':
          useDriver.getState().refreshOffers()
          toast.info('عرض جديد متاح!')
          if (navigator.vibrate) navigator.vibrate(200)
          break
        case 'order.status_changed':
          useDriver.getState().refreshActiveOrder()
          break
        case 'driver.status_changed':
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
  // CRITICAL: Wait for isInitialized before checking isAuthenticated.
  // Without this, the route guard fires BEFORE initialize() has a chance
  // to restore the session from localStorage, causing an immediate redirect
  // to /login even when the user just logged in.
  useEffect(() => {
    if (!isInitialized) return  // ← wait for initialize() to complete
    if (!isAuthenticated) {
      router.replace('/login')
      return
    }
    setBootChecked(true)
    fetchDriver()
  }, [isInitialized, isAuthenticated, router, fetchDriver])

  // ===== WebSocket subscribe =====
  useEffect(() => {
    if (!isConnected || !userID) return
    subscribe(`user:${userID}`)
    if (driver?.id) subscribe(`driver:${driver.id}`)
    if (driver?.current_order_id) subscribe(`order:${driver.current_order_id}`)
  }, [isConnected, userID, driver, subscribe])

  // ===== Initialize Mapbox =====
  useEffect(() => {
    if (!bootChecked || !mapContainerRef.current || mapRef.current) return

    let cancelled = false

    loadMapbox().then((mbgl: any) => {
      if (cancelled || !mapContainerRef.current) return

      try {
        mbgl.accessToken = MAPBOX_TOKEN
        const map = new mbgl.Map({
          container: mapContainerRef.current,
          style: 'mapbox://styles/mapbox/streets-v12',
          center: [31.2357, 30.0444],
          zoom: 13,
          attributionControl: false,
        })

        map.on('error', (e: any) => {
          console.error('Mapbox error:', e?.error?.message || e)
        })

        map.addControl(new mbgl.NavigationControl(), 'top-left')

        const loadTimeout = setTimeout(() => {
          if (!cancelled) {
            console.warn('Map timeout — showing anyway')
            setMapReady(true)
          }
        }, 8000)

        map.on('load', () => {
          clearTimeout(loadTimeout)
          if (cancelled) return
          setMapReady(true)
          setMapError(null)
          setTimeout(() => map.resize(), 200)
          if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition(
              (pos) => map.flyTo({ center: [pos.coords.longitude, pos.coords.latitude], zoom: 14 }),
              () => {},
              { enableHighAccuracy: true, timeout: 5000 }
            )
          }
        })

        mapRef.current = map

        const handleResize = () => mapRef.current?.resize()
        window.addEventListener('resize', handleResize)

        // Cleanup is handled by the outer return below
      } catch (err: any) {
        console.error('Map init error:', err)
        setMapError(err.message)
      }
    }).catch((err) => {
      console.error('Mapbox load failed:', err)
      setMapError('فشل تحميل الخريطة')
    })

    return () => {
      cancelled = true
      if (mapRef.current) {
        mapRef.current.remove()
        mapRef.current = null
      }
    }
  }, [bootChecked])

  // ===== Auto-refresh offers =====
  useEffect(() => {
    if (!driver || driver.status !== 'online') return
    const interval = setInterval(() => useDriver.getState().refreshOffers(), 10000)
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
      {/* Map container */}
      <div ref={mapContainerRef} className="absolute inset-0" />

      {/* Map loading/error overlay */}
      {(!mapReady || mapError) && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100 z-0">
          {mapError ? (
            <div className="text-center px-6">
              <p className="text-sm text-red-500 mb-3">⚠️ {mapError}</p>
              <button onClick={() => window.location.reload()} className="text-sm text-blue-500 underline">
                إعادة المحاولة
              </button>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
              <p className="text-sm text-gray-400">جاري تحميل الخريطة...</p>
            </div>
          )}
        </div>
      )}

      {/* Top Bar */}
      <div
        className="absolute top-0 left-0 right-0 z-10 flex items-center justify-between px-4 h-14"
        style={{ backgroundColor: 'rgba(91, 192, 222, 0.95)', paddingTop: 'env(safe-area-inset-top, 0px)' }}
      >
        <button
          onClick={() => router.push('/support')}
          className="w-9 h-9 rounded-full bg-white/20 flex items-center justify-center text-white"
        >
          <Headphones className="w-5 h-5" />
        </button>
        <div className="flex items-center gap-2">
          <span className="text-white font-medium text-sm">المندوب</span>
          <div className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-green-400' : 'bg-gray-400'} animate-pulse`} />
        </div>
        <button
          onClick={() => setDrawerOpen(true)}
          className="w-9 h-9 rounded-full bg-white/20 flex items-center justify-center text-white"
        >
          <Menu className="w-5 h-5" />
        </button>
      </div>

      {/* Recenter button */}
      <button
        onClick={() => {
          if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition((pos) => {
              mapRef.current?.flyTo({ center: [pos.coords.longitude, pos.coords.latitude], zoom: 15 })
            })
          }
        }}
        className="absolute bottom-32 left-4 z-10 w-10 h-10 rounded-full bg-white shadow-lg flex items-center justify-center"
      >
        <Navigation className="w-5 h-5 text-gray-700" />
      </button>

      {/* Online/Offline toggle */}
      <button
        onClick={handleToggleOnline}
        disabled={togglingOnline || !driver}
        className="absolute bottom-32 right-4 z-10 flex items-center gap-2 px-4 h-10 rounded-full shadow-lg font-medium text-sm transition-all active:scale-95 disabled:opacity-50"
        style={{ backgroundColor: isOnline ? '#FF6B35' : '#fff', color: isOnline ? '#fff' : '#333' }}
      >
        {togglingOnline ? <Loader2 className="w-4 h-4 animate-spin" /> : <Power className="w-4 h-4" />}
        {isOnline ? 'متصل' : 'غير متصل'}
      </button>

      {/* Bottom Card */}
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
                {!driver ? 'جاري تحميل البيانات...' : isOnline ? 'لا يوجد طلبات حالياً' : 'أنت غير متصل'}
              </p>
              <button onClick={() => setBottomCardExpanded(!bottomCardExpanded)} className="text-gray-400">
                <ChevronDown className={`w-5 h-5 transition-transform ${bottomCardExpanded ? 'rotate-180' : ''}`} />
              </button>
            </div>
            <p className="text-gray-500 text-xs">
              {!driver
                ? 'يرجى الانتظار...'
                : isOnline
                  ? 'يمكنك الانتظار للحصول على طلب جديد'
                  : 'اضغط على زر "متصل" للبدء في استقبال الطلبات'}
            </p>

            {error && (
              <div className="mt-2 text-xs text-red-500 bg-red-50 p-2 rounded-lg">⚠️ {error}</div>
            )}

            {bottomCardExpanded && driver && (
              <div className="mt-3 pt-3 border-t border-gray-100 space-y-2">
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Clock className="w-4 h-4" />
                  <span>الشيفت: {new Date().toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })}</span>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Package className="w-4 h-4" />
                  <span>طلبات اليوم: {driver?.total_deliveries || 0}</span>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <User className="w-4 h-4" />
                  <span>التقييم: {driver?.rating?.toFixed(1) || '5.0'} ⭐</span>
                </div>
              </div>
            )}
          </motion.div>
        )}
      </div>

      {/* Offer Modal */}
      <AnimatePresence>
        {activeOffer && offers.find((o) => o.id === activeOffer) && (
          <OfferModal offer={offers.find((o) => o.id === activeOffer)!} onClose={() => setActiveOffer(null)} />
        )}
      </AnimatePresence>

      {/* Side Drawer */}
      <SideDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />

      {/* Connection indicator */}
      {!isConnected && isOnline && (
        <div className="absolute top-16 left-1/2 -translate-x-1/2 z-20 bg-yellow-100 text-yellow-800 text-xs px-3 py-1 rounded-full shadow">
          جاري إعادة الاتصال...
        </div>
      )}
    </div>
  )
}
