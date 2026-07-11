import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  Bike, Phone, Lock, Loader2, Eye, EyeOff, AlertCircle,
  Power, Package, Star, LogOut, User,
} from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { toast } from 'sonner'

export default function DriverPage() {
  const { isAuthenticated, userID, login, logout, initialize } = useAuth()
  const { driver, fetchDriver, setOnline, setOffline, error } = useDriver()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [authError, setAuthError] = useState('')
  const [toggling, setToggling] = useState(false)
  const [driverLoaded, setDriverLoaded] = useState(false)

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

  // Reset driverLoaded when user logs out
  useEffect(() => {
    if (!isAuthenticated) {
      setDriverLoaded(false)
    }
  }, [isAuthenticated])

  // ===== Login handler =====
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setAuthError('')
    if (!phone || !password) {
      setAuthError('ادخل رقم الهاتف وكلمة المرور')
      return
    }
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
      setToggling(false)
    }
  }

  // ===== Logout =====
  const handleLogout = () => {
    logout()
    setDriverLoaded(false)
    setPhone('')
    setPassword('')
    toast.success('تم تسجيل الخروج')
  }

  // ===== LOGIN SCREEN =====
  if (!isAuthenticated) {
    return (
      <div className="min-h-dvh bg-white flex flex-col" dir="rtl">
        <div className="flex-1 flex flex-col items-center justify-center px-6">
          <motion.div
            initial={{ opacity: 0, scale: 0.85 }}
            animate={{ opacity: 1, scale: 1 }}
            className="w-20 h-20 rounded-2xl flex items-center justify-center mb-5 shadow-lg"
            style={{ backgroundColor: '#FF6B35' }}
          >
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
              <Phone className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
              <input
                type="tel"
                dir="ltr"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="01xxxxxxxxx"
                autoComplete="tel"
                className="w-full h-12 pr-10 pl-4 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500"
              />
            </div>
            <div className="relative">
              <Lock className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="كلمة المرور"
                autoComplete="current-password"
                className="w-full h-12 pr-10 pl-10 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <button
              type="submit"
              className="w-full h-12 rounded-lg font-medium text-white transition-colors flex items-center justify-center gap-2"
              style={{ backgroundColor: '#FF6B35' }}
            >
              تسجيل الدخول
            </button>
          </form>

          <div className="mt-6 text-center text-xs text-gray-400">
            <p>حساب تجريبي:</p>
            <p dir="ltr" className="mt-1 font-mono">01012345678 / 12345678</p>
          </div>
        </div>
      </div>
    )
  }

  // ===== DASHBOARD (authenticated) =====
  const isOnline = driver?.status === 'online' || driver?.status === 'busy'

  return (
    <div className="min-h-dvh bg-gray-50" dir="rtl">
      {/* Header */}
      <header className="sticky top-0 z-30 bg-white border-b border-gray-200 h-14 flex items-center justify-between px-4">
        <div className="flex items-center gap-2">
          <Bike className="w-5 h-5 text-orange-500" />
          <span className="font-bold">AVEX Driver</span>
        </div>
        <button
          onClick={handleLogout}
          className="flex items-center gap-1 text-sm text-gray-600 hover:text-red-500"
        >
          <LogOut className="w-4 h-4" /> خروج
        </button>
      </header>

      <main className="p-4 max-w-2xl mx-auto space-y-4">
        {/* Status card */}
        <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="text-xl font-bold text-gray-900">
                {isOnline ? '🟢 متصل' : '🔴 غير متصل'}
              </h1>
              <p className="text-sm text-gray-500 mt-1">
                {isOnline ? 'جاهز لاستقبال الطلبات' : 'اضغط للبدء'}
              </p>
            </div>
            <button
              onClick={handleToggle}
              disabled={toggling || !driver}
              className="flex items-center gap-2 px-5 h-11 rounded-xl font-medium text-white transition-all active:scale-95 disabled:opacity-50"
              style={{ backgroundColor: isOnline ? '#FF6B35' : '#10B981' }}
            >
              {toggling ? <Loader2 className="w-4 h-4 animate-spin" /> : <Power className="w-4 h-4" />}
              {isOnline ? 'إيقاف' : 'ابدأ'}
            </button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-white rounded-xl p-4 shadow-sm border border-gray-100 flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-orange-50 flex items-center justify-center">
              <Package className="w-5 h-5 text-orange-500" />
            </div>
            <div>
              <p className="text-xs text-gray-500">طلبات اليوم</p>
              <p className="text-lg font-bold">{driver?.total_deliveries ?? 0}</p>
            </div>
          </div>
          <div className="bg-white rounded-xl p-4 shadow-sm border border-gray-100 flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-orange-50 flex items-center justify-center">
              <Star className="w-5 h-5 text-orange-500" />
            </div>
            <div>
              <p className="text-xs text-gray-500">التقييم</p>
              <p className="text-lg font-bold">{(driver?.rating ?? 5).toFixed(1)} ⭐</p>
            </div>
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800">خطأ</p>
              <p className="text-xs text-red-600 mt-1">{error}</p>
            </div>
          </div>
        )}

        {/* No driver data warning */}
        {!driver && driverLoaded && (
          <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-4 flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-yellow-800">بيانات المندوب غير متاحة</p>
              <p className="text-xs text-yellow-700 mt-1">
                قد لا يكون لديك سجل في جدول المندوبين. تواصل مع الإدارة لتفعيل حسابك.
              </p>
            </div>
          </div>
        )}

        {/* Driver info */}
        {driver && (
          <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
            <h2 className="font-bold text-gray-900 mb-3 flex items-center gap-2">
              <User className="w-4 h-4" /> معلومات المندوب
            </h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between py-2 border-b border-gray-100">
                <span className="text-gray-500">المعرف</span>
                <span className="font-mono" dir="ltr">{driver.id}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-100">
                <span className="text-gray-500">المركبة</span>
                <span>{driver.vehicle_type}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-100">
                <span className="text-gray-500">رقم اللوحة</span>
                <span dir="ltr">{driver.license_plate}</span>
              </div>
              <div className="flex justify-between py-2">
                <span className="text-gray-500">معدل القبول</span>
                <span>{driver.acceptance_rate}%</span>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
