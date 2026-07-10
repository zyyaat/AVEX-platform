import { useEffect } from 'react'
import { useRouter } from '@/lib/navigation'
import { User, Bike, Star, Settings, LogOut } from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { BottomTabBar } from '@/components/BottomTabBar'
import { toast } from 'sonner'

export default function ProfilePage() {
  const router = useRouter()
  const { isAuthenticated, logout, userID } = useAuth()
  const { driver, fetchDriver } = useDriver()

  useEffect(() => {
    if (!isAuthenticated) { router.replace('/login'); return }
    fetchDriver()
  }, [isAuthenticated, router, fetchDriver])

  const handleLogout = () => {
    logout()
    useDriver.getState().clear()
    toast.success('تم تسجيل الخروج')
    router.replace('/login')
  }

  return (
    <div className="min-h-dvh bg-gray-50 pb-16" dir="rtl">
      {/* Header */}
      <div className="px-5 py-6 text-white" style={{ backgroundColor: '#FF6B35' }}>
        <div className="flex items-center gap-3">
          <div className="w-16 h-16 rounded-full bg-white/20 flex items-center justify-center">
            <User className="w-8 h-8 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-bold">المندوب</h1>
            <p className="text-white/80 text-sm">#{userID?.slice(0, 8) || '---'}</p>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="p-4 grid grid-cols-3 gap-3">
        <div className="bg-white rounded-2xl p-4 text-center">
          <p className="text-2xl font-bold">{driver?.total_deliveries || 0}</p>
          <p className="text-xs text-gray-500">توصيلة</p>
        </div>
        <div className="bg-white rounded-2xl p-4 text-center">
          <p className="text-2xl font-bold">{driver?.rating?.toFixed(1) ?? "5.0" || '5.0'}</p>
          <p className="text-xs text-gray-500">تقييم</p>
        </div>
        <div className="bg-white rounded-2xl p-4 text-center">
          <p className="text-2xl font-bold">{driver?.acceptance_rate || 100}%</p>
          <p className="text-xs text-gray-500">قبول</p>
        </div>
      </div>

      {/* Vehicle info */}
      {driver && (
        <div className="px-4">
          <div className="bg-white rounded-2xl p-4 flex items-center gap-3">
            <Bike className="w-6 h-6 text-gray-500" />
            <div>
              <p className="text-sm font-medium">
                {driver.vehicle_type === 'bike' ? 'دراجة' : driver.vehicle_type === 'car' ? 'سيارة' : 'سكوتر'}
              </p>
              <p className="text-xs text-gray-500">{driver.license_plate}</p>
            </div>
          </div>
        </div>
      )}

      {/* Menu */}
      <div className="p-4 space-y-2">
        <button className="w-full bg-white rounded-2xl p-4 flex items-center gap-3">
          <Settings className="w-5 h-5 text-gray-500" />
          <span className="text-sm">الإعدادات</span>
        </button>
        <button onClick={handleLogout} className="w-full bg-white rounded-2xl p-4 flex items-center gap-3 text-red-500">
          <LogOut className="w-5 h-5" />
          <span className="text-sm font-medium">تسجيل الخروج</span>
        </button>
      </div>

      <BottomTabBar />
    </div>
  )
}
