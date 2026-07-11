import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Bike, Phone, Lock, Loader2, Eye, EyeOff, AlertCircle,
  Power, Package, Star, LogOut, User, Wallet, Clock,
  Store, MapPin, Navigation, CheckCircle2, X, ChevronDown,
  TrendingUp, ArrowLeft, Home, Map as MapIcon, Headphones,
} from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { toast } from 'sonner'

type Tab = 'home' | 'earnings' | 'history' | 'profile'

export default function DriverPage() {
  const { isAuthenticated, userID, login, logout, initialize } = useAuth()
  const {
    driver, offers, activeOrder, orderHistory, wallet, transactions, error,
    fetchDriver, setOnline, setOffline, refreshOffers, refreshActiveOrder,
    refreshHistory, refreshWallet, acceptOffer, rejectOffer,
    markPickedUp, markDelivered,
  } = useDriver()

  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [authError, setAuthError] = useState('')
  const [toggling, setToggling] = useState(false)
  const [driverLoaded, setDriverLoaded] = useState(false)
  const [tab, setTab] = useState<Tab>('home')
  const [activeOfferId, setActiveOfferId] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [mapReady, setMapReady] = useState(false)
  const [mapError, setMapError] = useState<string | null>(null)
  const mapContainerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<any>(null)

  // Restore session on mount
  useEffect(() => {
    initialize()
  }, [])

  // Fetch driver data when authenticated
  useEffect(() => {
    if (isAuthenticated && userID && !driverLoaded) {
      fetchDriver().finally(() => setDriverLoaded(true))
    }
  }, [isAuthenticated, userID, driverLoaded, fetchDriver])

  // Reset on logout
  useEffect(() => {
    if (!isAuthenticated) { setDriverLoaded(false); setTab('home') }
  }, [isAuthenticated])

  // Auto-refresh offers + active order when online
  useEffect(() => {
    if (!driver || (driver.status !== 'online' && driver.status !== 'busy')) return
    refreshOffers()
    refreshActiveOrder()
    const interval = setInterval(() => {
      refreshOffers()
      refreshActiveOrder()
    }, 10000)
    return () => clearInterval(interval)
  }, [driver?.status, refreshOffers, refreshActiveOrder])

  // Auto-show offer modal
  useEffect(() => {
    if (offers.length > 0 && !activeOfferId && !activeOrder) {
      setActiveOfferId(offers[0].id)
    }
  }, [offers, activeOfferId, activeOrder])

  // Fetch tab data on switch
  useEffect(() => {
    if (tab === 'earnings') refreshWallet()
    if (tab === 'history') refreshHistory()
  }, [tab, refreshWallet, refreshHistory])

  // ===== Map: Leaflet (lightweight, fast, no token needed) =====
  useEffect(() => {
    if (tab !== 'home' || !isAuthenticated) return
    if (mapRef.current || !mapContainerRef.current) return

    let cancelled = false

    // Load Leaflet CSS (tiny — 14KB)
    if (!document.querySelector('#leaflet-css')) {
      const link = document.createElement('link')
      link.id = 'leaflet-css'
      link.rel = 'stylesheet'
      link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css'
      document.head.appendChild(link)
    }

    // Load Leaflet JS (tiny — 40KB vs Mapbox 200KB) and init map
    const initMap = (L: any) => {
      if (cancelled || !mapContainerRef.current) return
      try {
        const map = L.map(mapContainerRef.current, {
          center: [30.0444, 31.2357], // Cairo [lat, lng]
          zoom: 13,
          zoomControl: true,
          attributionControl: false,
        })

        // Use free CARTO tiles (no token, fast CDN, light theme)
        L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
          maxZoom: 19,
          subdomains: 'abcd',
        }).addTo(map)

        // Add driver marker
        const driverMarker = L.marker([30.0444, 31.2357]).addTo(map)
        driverMarker.bindPopup('موقعك الحالي').openPopup()

        mapRef.current = map
        setMapReady(true)
        setMapError(null)

        // Try to get user location and move map
        if (navigator.geolocation) {
          navigator.geolocation.getCurrentPosition(
            (pos) => {
              if (cancelled || !mapRef.current) return
              const lat = pos.coords.latitude
              const lng = pos.coords.longitude
              mapRef.current.setView([lat, lng], 14)
              driverMarker.setLatLng([lat, lng])
            },
            () => {},
            { enableHighAccuracy: true, timeout: 5000 }
          )
        }

        // Fix size after render
        setTimeout(() => {
          if (!cancelled && mapRef.current) {
            mapRef.current.invalidateSize()
          }
        }, 100)
      } catch (err: any) {
        console.error('Leaflet init error:', err)
        setMapError(err.message || 'فشل تحميل الخريطة')
      }
    }

    if ((window as any).L) {
      initMap((window as any).L)
    } else {
      const script = document.createElement('script')
      script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js'
      script.onload = () => {
        if (!cancelled && (window as any).L) {
          initMap((window as any).L)
        }
      }
      script.onerror = () => {
        if (!cancelled) setMapError('فشل تحميل الخريطة')
      }
      document.head.appendChild(script)
    }

    return () => {
      cancelled = true
      if (mapRef.current) {
        mapRef.current.remove()
        mapRef.current = null
      }
      setMapReady(false)
    }
  }, [tab, isAuthenticated])

  // ===== Login =====
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setAuthError('')
    if (!phone || !password) { setAuthError('ادخل رقم الهاتف وكلمة المرور'); return }
    try {
      await login(phone, password)
      toast.success('تم تسجيل الدخول بنجاح')
    } catch (err: any) {
      setAuthError(err.message || 'فشل تسجيل الدخول')
    }
  }

  // ===== Toggle online/offline =====
  const handleToggle = async () => {
    setToggling(true)
    try {
      if (driver?.status === 'online') { await setOffline(); toast.success('أنت الآن غير متصل') }
      else { await setOnline(); toast.success('أنت الآن متصل — بانتظار الطلبات') }
    } catch (err: any) { toast.error(err.message || 'فشل تغيير الحالة') }
    finally { setToggling(false) }
  }

  // ===== Accept/Reject offer =====
  const handleAccept = async () => {
    if (!activeOfferId) return
    setBusy('accept')
    try { await acceptOffer(activeOfferId); toast.success('تم قبول الطلب!'); setActiveOfferId(null) }
    catch (err: any) { toast.error(err.message) }
    finally { setBusy(null) }
  }
  const handleReject = async () => {
    if (!activeOfferId) return
    setBusy('reject')
    try { await rejectOffer(activeOfferId); setActiveOfferId(null) }
    catch (err: any) { toast.error(err.message) }
    finally { setBusy(null) }
  }

  // ===== Pickup/Deliver =====
  const handlePickup = async () => {
    if (!activeOrder) return
    setBusy('pickup')
    try { await markPickedUp(activeOrder.id); toast.success('تم استلام الطلب') }
    catch (err: any) { toast.error(err.message) }
    finally { setBusy(null) }
  }
  const handleDeliver = async () => {
    if (!activeOrder) return
    setBusy('deliver')
    try { await markDelivered(activeOrder.id); toast.success('تم التوصيل بنجاح! 🎉') }
    catch (err: any) { toast.error(err.message) }
    finally { setBusy(null) }
  }

  // ===== Logout =====
  const handleLogout = () => {
    logout()
    setDriverLoaded(false); setPhone(''); setPassword(''); setTab('home')
    toast.success('تم تسجيل الخروج')
  }

  // ===== LOGIN SCREEN =====
  if (!isAuthenticated) {
    return (
      <div className="min-h-dvh bg-white flex flex-col" dir="rtl">
        <div className="flex-1 flex flex-col items-center justify-center px-6">
          <motion.div initial={{ opacity: 0, scale: 0.85 }} animate={{ opacity: 1, scale: 1 }}
            className="w-20 h-20 rounded-2xl flex items-center justify-center mb-5 shadow-lg"
            style={{ backgroundColor: '#FF6B35' }}>
            <Bike className="w-10 h-10 text-white" strokeWidth={2.5} />
          </motion.div>
          <h1 className="text-2xl font-bold mb-1">AVEX Driver</h1>
          <p className="text-sm text-gray-500 mb-8 text-center">تطبيق المندوب — للمندوبين المعتمدين</p>
          <form onSubmit={handleLogin} className="w-full max-w-sm space-y-3" noValidate>
            {authError && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-3 flex items-start gap-2 text-sm">
                <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
                <span className="text-red-700">{authError}</span>
              </div>
            )}
            <div className="relative">
              <Phone className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input type="tel" dir="ltr" value={phone} onChange={(e) => setPhone(e.target.value)}
                placeholder="01xxxxxxxxx" autoComplete="tel"
                className="w-full h-12 pr-10 pl-4 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500" />
            </div>
            <div className="relative">
              <Lock className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input type={showPassword ? 'text' : 'password'} value={password} onChange={(e) => setPassword(e.target.value)}
                placeholder="كلمة المرور" autoComplete="current-password"
                className="w-full h-12 pr-10 pl-10 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500" />
              <button type="button" onClick={() => setShowPassword(!showPassword)}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700">
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <button type="submit" className="w-full h-12 rounded-lg font-medium text-white flex items-center justify-center gap-2"
              style={{ backgroundColor: '#FF6B35' }}>تسجيل الدخول</button>
          </form>
          <div className="mt-6 text-center text-xs text-gray-400">
            <p>حساب تجريبي:</p><p dir="ltr" className="mt-1 font-mono">01012345678 / 12345678</p>
          </div>
        </div>
      </div>
    )
  }

  // ===== DASHBOARD =====
  const isOnline = driver?.status === 'online' || driver?.status === 'busy'
  const currentOffer = offers.find(o => o.id === activeOfferId)

  return (
    <div className="min-h-dvh bg-gray-50" dir="rtl">
      {/* Header */}
      <header className="sticky top-0 z-30 bg-white border-b border-gray-200 h-14 flex items-center justify-between px-4">
        <div className="flex items-center gap-2">
          <Bike className="w-5 h-5 text-orange-500" />
          <span className="font-bold">AVEX Driver</span>
        </div>
        <button onClick={handleLogout} className="flex items-center gap-1 text-sm text-gray-600 hover:text-red-500">
          <LogOut className="w-4 h-4" /> خروج
        </button>
      </header>

      {/* Tab Bar */}
      <div className="sticky top-14 z-20 bg-white border-b border-gray-200 flex">
        {[
          { id: 'home' as Tab, label: 'الرئيسية', icon: Home },
          { id: 'earnings' as Tab, label: 'الأرباح', icon: Wallet },
          { id: 'history' as Tab, label: 'السجل', icon: Clock },
          { id: 'profile' as Tab, label: 'الملف', icon: User },
        ].map(({ id, label, icon: Icon }) => (
          <button key={id} onClick={() => setTab(id)}
            className={`flex-1 flex flex-col items-center gap-0.5 py-2 text-xs transition-colors ${tab === id ? 'text-orange-500 font-bold' : 'text-gray-500'}`}>
            <Icon className="w-5 h-5" />{label}
          </button>
        ))}
      </div>

      <main className="max-w-2xl mx-auto pb-20">
        {/* ===== HOME TAB — Talabat Rider style ===== */}
        {tab === 'home' && (
          <div className="relative">
            {/* Full-screen map */}
            <div className="relative w-full" style={{ height: 'calc(100dvh - 112px)' }}>
              <div ref={mapContainerRef} className="absolute inset-0" />
              {/* Map loading/error overlay */}
              {(!mapReady || mapError) && (
                <div className="absolute inset-0 flex items-center justify-center bg-gray-100 z-10">
                  {mapError ? (
                    <div className="text-center px-6">
                      <MapIcon className="w-8 h-8 text-gray-300 mx-auto mb-2" />
                      <p className="text-sm text-gray-500">{mapError}</p>
                    </div>
                  ) : (
                    <div className="flex flex-col items-center gap-2">
                      <Loader2 className="w-6 h-6 animate-spin text-gray-400" />
                      <p className="text-xs text-gray-400">جاري تحميل الخريطة...</p>
                    </div>
                  )}
                </div>
              )}

              {/* Top floating bar (over map) */}
              <div className="absolute top-3 left-3 right-3 z-20 flex items-center justify-between">
                <button className="w-10 h-10 rounded-full bg-white shadow-lg flex items-center justify-center">
                  <Headphones className="w-5 h-5 text-gray-700" />
                </button>
                <div className="bg-white/95 backdrop-blur px-4 py-1.5 rounded-full shadow-lg flex items-center gap-2">
                  <div className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`} />
                  <span className="text-xs font-bold">{isOnline ? 'متصل' : 'غير متصل'}</span>
                </div>
                <button onClick={handleLogout} className="w-10 h-10 rounded-full bg-white shadow-lg flex items-center justify-center">
                  <LogOut className="w-5 h-5 text-gray-700" />
                </button>
              </div>

              {/* Side buttons (right) */}
              {mapReady && (
                <div className="absolute right-3 bottom-24 z-20 flex flex-col gap-2">
                  <button
                    onClick={() => {
                      if (navigator.geolocation && mapRef.current) {
                        navigator.geolocation.getCurrentPosition((pos) => {
                          mapRef.current?.setView([pos.coords.latitude, pos.coords.longitude], 15)
                        })
                      }
                    }}
                    className="w-10 h-10 rounded-full bg-white shadow-lg flex items-center justify-center"
                  >
                    <Navigation className="w-5 h-5 text-gray-700" />
                  </button>
                </div>
              )}

              {/* Online/Offline toggle button (right, above recenter) */}
              {mapReady && (
                <button onClick={handleToggle} disabled={toggling || !driver}
                  className="absolute right-3 bottom-4 z-20 w-10 h-10 rounded-full shadow-lg flex items-center justify-center disabled:opacity-50"
                  style={{ backgroundColor: isOnline ? '#FF6B35' : '#10B981' }}>
                  {toggling ? <Loader2 className="w-5 h-5 animate-spin text-white" /> : <Power className="w-5 h-5 text-white" />}
                </button>
              )}
            </div>

            {/* Floating bottom card (over map) — Talabat style */}
            <div className="absolute bottom-0 left-0 right-0 z-30 bg-white rounded-t-2xl shadow-2xl px-5 py-4 pb-6"
              style={{ paddingBottom: 'calc(1.5rem + env(safe-area-inset-bottom, 0px))' }}>
              {/* Active Order */}
              {activeOrder ? (
                <ActiveOrderCard order={activeOrder} busy={busy} onPickup={handlePickup} onDeliver={handleDeliver} />
              ) : (
                <>
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-gray-800 text-sm font-medium">
                      {!driver ? 'جاري تحميل البيانات...' : isOnline ? 'لا يوجد طلبات حالياً' : 'أنت غير متصل'}
                    </p>
                    {!mapReady && (
                      <button onClick={handleToggle} disabled={toggling || !driver}
                        className="flex items-center gap-2 px-4 h-9 rounded-full font-medium text-white text-sm disabled:opacity-50"
                        style={{ backgroundColor: isOnline ? '#FF6B35' : '#10B981' }}>
                        {toggling ? <Loader2 className="w-4 h-4 animate-spin" /> : <Power className="w-4 h-4" />}
                        {isOnline ? 'إيقاف' : 'ابدأ'}
                      </button>
                    )}
                  </div>
                  <p className="text-gray-500 text-xs">
                    {!driver ? 'يرجى الانتظار...' : isOnline ? 'يمكنك الانتظار للحصول على طلب جديد' : 'اضغط على زر "ابدأ" للاتصال واستقبال الطلبات'}
                  </p>

                  {/* Stats row */}
                  {driver && (
                    <div className="flex items-center gap-4 mt-3 pt-3 border-t border-gray-100">
                      <div className="flex items-center gap-1.5 text-xs text-gray-500">
                        <Package className="w-4 h-4" />
                        <span>{driver.total_deliveries ?? 0} توصيلة</span>
                      </div>
                      <div className="flex items-center gap-1.5 text-xs text-gray-500">
                        <Star className="w-4 h-4" />
                        <span>{(driver.rating ?? 5).toFixed(1)} ⭐</span>
                      </div>
                      <div className="flex items-center gap-1.5 text-xs text-gray-500">
                        <Bike className="w-4 h-4" />
                        <span>{driver.vehicle_type}</span>
                      </div>
                    </div>
                  )}

                  {/* Error */}
                  {error && (
                    <div className="mt-2 text-xs text-red-500 bg-red-50 p-2 rounded-lg">⚠️ {error}</div>
                  )}

                  {/* No driver data */}
                  {!driver && driverLoaded && (
                    <div className="mt-2 text-xs text-yellow-700 bg-yellow-50 p-2 rounded-lg">
                      ⚠️ بيانات المندوب غير متاحة — تواصل مع الإدارة
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        )}

        {/* ===== EARNINGS TAB ===== */}
        {tab === 'earnings' && (
          <div className="space-y-4">
            <div className="bg-gradient-to-l from-orange-500 to-orange-600 rounded-xl p-6 text-white shadow-lg">
              <p className="text-sm opacity-90">رصيد المحفظة</p>
              <p className="text-3xl font-bold mt-1">
                {wallet ? `${(wallet.balance / 100).toFixed(2)} ج.م` : '—'}
              </p>
              <p className="text-xs opacity-75 mt-2">
                pending: {wallet ? `${(wallet.pending_balance / 100).toFixed(2)} ج.م` : '—'}
              </p>
            </div>
            <div className="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
              <h3 className="font-bold mb-3 flex items-center gap-2"><TrendingUp className="w-4 h-4" /> آخر المعاملات</h3>
              {transactions.length === 0 ? (
                <p className="text-sm text-gray-400 text-center py-4">لا توجد معاملات</p>
              ) : (
                <div className="space-y-2">
                  {transactions.slice(0, 20).map((tx: any) => (
                    <div key={tx.id} className="flex justify-between items-center py-2 border-b border-gray-100">
                      <div>
                        <p className="text-sm font-medium">{tx.category || tx.type}</p>
                        <p className="text-xs text-gray-400">{new Date(tx.created_at).toLocaleDateString('ar')}</p>
                      </div>
                      <span className={`text-sm font-bold ${tx.type === 'credit' ? 'text-green-600' : 'text-red-500'}`}>
                        {tx.type === 'credit' ? '+' : '-'}{(tx.amount / 100).toFixed(2)} ج.م
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* ===== HISTORY TAB ===== */}
        {tab === 'history' && (
          <div className="space-y-3">
            <h2 className="font-bold text-lg">سجل الطلبات</h2>
            {orderHistory.length === 0 ? (
              <div className="bg-white rounded-xl p-8 text-center border border-gray-100">
                <Package className="w-12 h-12 text-gray-300 mx-auto mb-2" />
                <p className="text-sm text-gray-400">لا توجد طلبات سابقة</p>
              </div>
            ) : (
              orderHistory.map((order: any) => (
                <div key={order.id} className="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <p className="font-medium text-sm" dir="ltr">{order.order_number || order.id}</p>
                      <p className="text-xs text-gray-400">{new Date(order.created_at).toLocaleString('ar')}</p>
                    </div>
                    <StatusBadge status={order.status} />
                  </div>
                  <div className="flex items-center gap-2 text-xs text-gray-500">
                    <Store className="w-3 h-3" /> {order.restaurant_name || 'مطعم'}
                    <span>•</span>
                    <span>{(order.total / 100).toFixed(2)} ج.م</span>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {/* ===== PROFILE TAB ===== */}
        {tab === 'profile' && (
          <div className="space-y-4">
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-16 h-16 rounded-full bg-orange-100 flex items-center justify-center">
                  <Bike className="w-8 h-8 text-orange-500" />
                </div>
                <div>
                  <h2 className="font-bold text-lg">مندوب AVEX</h2>
                  <p className="text-sm text-gray-500" dir="ltr">{driver?.id || '—'}</p>
                </div>
              </div>
            </div>
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h3 className="font-bold mb-3">المعلومات</h3>
              <div className="space-y-2 text-sm">
                <InfoRow label="المركبة" value={driver?.vehicle_type || '—'} />
                <InfoRow label="رقم اللوحة" value={driver?.license_plate || '—'} />
                <InfoRow label="الحالة" value={driver?.status || '—'} />
                <InfoRow label="إجمالي التوصيلات" value={driver?.total_deliveries ?? 0} />
                <InfoRow label="معدل القبول" value={`${driver?.acceptance_rate ?? 100}%`} />
                <InfoRow label="معدل الإكمال" value={`${driver?.completion_rate ?? 100}%`} />
                <InfoRow label="التقييم" value={`${(driver?.rating ?? 5).toFixed(1)} ⭐ (${driver?.rating_count ?? 0})`} />
              </div>
            </div>
            <button onClick={handleLogout}
              className="w-full h-12 rounded-xl border border-red-200 text-red-600 font-medium flex items-center justify-center gap-2 hover:bg-red-50">
              <LogOut className="w-4 h-4" /> تسجيل الخروج
            </button>
          </div>
        )}
      </main>

      {/* ===== OFFER MODAL ===== */}
      <AnimatePresence>
        {currentOffer && !activeOrder && (
          <OfferModal
            offer={currentOffer}
            busy={busy}
            onAccept={handleAccept}
            onReject={handleReject}
          />
        )}
      </AnimatePresence>
    </div>
  )
}

// ===== Helper Components =====

function StatCard({ icon: Icon, label, value }: { icon: any; label: string; value: any }) {
  return (
    <div className="bg-white rounded-xl p-4 shadow-sm border border-gray-100 flex items-center gap-3">
      <div className="w-10 h-10 rounded-lg bg-orange-50 flex items-center justify-center">
        <Icon className="w-5 h-5 text-orange-500" />
      </div>
      <div><p className="text-xs text-gray-500">{label}</p><p className="text-lg font-bold">{value}</p></div>
    </div>
  )
}

function ErrorBox({ message }: { message: string }) {
  return (
    <div className="bg-red-50 border border-red-200 rounded-xl p-4 flex items-start gap-3">
      <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
      <div><p className="text-sm font-medium text-red-800">خطأ</p><p className="text-xs text-red-600 mt-1">{message}</p></div>
    </div>
  )
}

function InfoRow({ label, value, mono }: { label: string; value: any; mono?: boolean }) {
  return (
    <div className="flex justify-between py-2 border-b border-gray-100">
      <span className="text-gray-500">{label}</span>
      <span className={mono ? 'font-mono' : ''} dir={mono ? 'ltr' : 'rtl'}>{value}</span>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const labels: Record<string, string> = {
    delivered: 'تم التوصيل', cancelled: 'ملغي', picked_up: 'تم الاستلام',
    assigned: 'مُسند', pending: 'جديد', preparing: 'قيد التحضير', ready: 'جاهز',
  }
  const colors: Record<string, string> = {
    delivered: 'bg-green-100 text-green-700', cancelled: 'bg-red-100 text-red-700',
    picked_up: 'bg-blue-100 text-blue-700', assigned: 'bg-orange-100 text-orange-700',
    pending: 'bg-gray-100 text-gray-700',
  }
  return (
    <span className={`text-xs px-2 py-1 rounded-full ${colors[status] || 'bg-gray-100 text-gray-700'}`}>
      {labels[status] || status}
    </span>
  )
}

function ActiveOrderCard({ order, busy, onPickup, onDeliver }: {
  order: any; busy: string | null; onPickup: () => void; onDeliver: () => void
}) {
  const [expanded, setExpanded] = useState(false)
  const statusLabels: Record<string, string> = {
    assigned: 'تم القبول — اذهب للمطعم', picked_up: 'تم الاستلام — اذهب للعميل',
    delivering: 'في الطريق للعميل',
  }
  return (
    <div className="bg-white rounded-xl p-5 shadow-sm border-2 border-orange-200">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-bold flex items-center gap-2"><Package className="w-4 h-4 text-orange-500" /> طلب نشط</h3>
        <StatusBadge status={order.status} />
      </div>
      <p className="text-sm text-gray-600 mb-3">{statusLabels[order.status] || order.status}</p>
      <div className="space-y-2 text-sm">
        <div className="flex items-center gap-2 text-gray-600">
          <Store className="w-4 h-4 text-gray-400" />
          <span>{order.restaurant_name || 'مطعم'}</span>
        </div>
        <div className="flex items-center gap-2 text-gray-600">
          <MapPin className="w-4 h-4 text-gray-400" />
          <span>{order.delivery_address || order.delivery_info?.address || 'عنوان التوصيل'}</span>
        </div>
        <div className="flex items-center gap-2 text-gray-600">
          <Phone className="w-4 h-4 text-gray-400" />
          <span dir="ltr">{order.customer_phone || '—'}</span>
        </div>
        <div className="flex items-center gap-2 text-gray-600">
          <span className="font-bold text-orange-600">{(order.total / 100).toFixed(2)} ج.م</span>
          <span className="text-gray-400">•</span>
          <span>{order.payment_method === 'cash' ? 'نقدي' : 'بطاقة'}</span>
        </div>
      </div>
      {expanded && order.items && (
        <div className="mt-3 pt-3 border-t border-gray-100 space-y-1">
          {order.items.map((item: any, i: number) => (
            <div key={i} className="flex justify-between text-xs text-gray-500">
              <span>{item.name_ar || item.name} ×{item.quantity}</span>
              <span>{(item.price / 100).toFixed(2)} ج.م</span>
            </div>
          ))}
        </div>
      )}
      <button onClick={() => setExpanded(!expanded)} className="mt-3 text-xs text-gray-400 flex items-center gap-1">
        <ChevronDown className={`w-4 h-4 transition-transform ${expanded ? 'rotate-180' : ''}`} />
        {expanded ? 'إخفاء التفاصيل' : 'عرض التفاصيل'}
      </button>
      {/* Action Buttons */}
      <div className="mt-4 flex gap-2">
        {order.status === 'assigned' && (
          <button onClick={onPickup} disabled={busy === 'pickup'}
            className="flex-1 h-11 rounded-xl font-medium text-white flex items-center justify-center gap-2 disabled:opacity-50"
            style={{ backgroundColor: '#FF6B35' }}>
            {busy === 'pickup' ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
            تم استلام الطلب
          </button>
        )}
        {order.status === 'picked_up' && (
          <button onClick={onDeliver} disabled={busy === 'deliver'}
            className="flex-1 h-11 rounded-xl font-medium text-white flex items-center justify-center gap-2 disabled:opacity-50"
            style={{ backgroundColor: '#10B981' }}>
            {busy === 'deliver' ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
            تم التوصيل
          </button>
        )}
      </div>
    </div>
  )
}

function OfferModal({ offer, busy, onAccept, onReject }: {
  offer: any; busy: string | null; onAccept: () => void; onReject: () => void
}) {
  const [secondsLeft, setSecondsLeft] = useState(15)
  useEffect(() => {
    const interval = setInterval(() => {
      setSecondsLeft((s) => Math.max(0, s - 1))
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/50 p-4">
      <motion.div initial={{ y: 50, opacity: 0 }} animate={{ y: 0, opacity: 1 }} exit={{ y: 50, opacity: 0 }}
        className="bg-white rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        {/* Timer */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-bold text-lg flex items-center gap-2">
            <Package className="w-5 h-5 text-orange-500" /> عرض طلب جديد!
          </h3>
          <div className={`text-2xl font-bold ${secondsLeft <= 5 ? 'text-red-500' : 'text-orange-500'}`}>
            {secondsLeft}s
          </div>
        </div>
        {/* Offer details */}
        <div className="space-y-3 mb-5">
          <div className="flex justify-between text-sm">
            <span className="text-gray-500">المسافة المقدرة</span>
            <span className="font-medium">{offer.est_distance_m ? `${(offer.est_distance_m / 1000).toFixed(1)} كم` : '—'}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-gray-500">المدة المقدرة</span>
            <span className="font-medium">{offer.est_duration_s ? `${Math.round(offer.est_duration_s / 60)} دقيقة` : '—'}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-gray-500">الأجر المقدر</span>
            <span className="font-bold text-orange-600">
              {offer.est_fare_cents ? `${(offer.est_fare_cents / 100).toFixed(2)} ج.م` : '—'}
            </span>
          </div>
        </div>
        {/* Buttons */}
        <div className="flex gap-2">
          <button onClick={onReject} disabled={busy === 'reject'}
            className="flex-1 h-12 rounded-xl border border-gray-200 text-gray-600 font-medium flex items-center justify-center gap-2 hover:bg-gray-50 disabled:opacity-50">
            {busy === 'reject' ? <Loader2 className="w-4 h-4 animate-spin" /> : <X className="w-4 h-4" />}
            رفض
          </button>
          <button onClick={onAccept} disabled={busy === 'accept'}
            className="flex-1 h-12 rounded-xl text-white font-bold flex items-center justify-center gap-2 disabled:opacity-50"
            style={{ backgroundColor: '#FF6B35' }}>
            {busy === 'accept' ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
            قبول
          </button>
        </div>
      </motion.div>
    </motion.div>
  )
}
