import { useState, useEffect } from 'react'
import { Bike, Package, Star, TrendingUp, Loader2, AlertCircle, Power } from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { toast } from 'sonner'

export default function DriverHome() {
  const { userID } = useAuth()
  const { driver, fetchDriver, setOnline, setOffline, error } = useDriver()
  const [loading, setLoading] = useState(true)
  const [toggling, setToggling] = useState(false)

  useEffect(() => {
    fetchDriver()
      .finally(() => setLoading(false))
  }, [fetchDriver])

  const handleToggleOnline = async () => {
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

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="w-6 h-6 animate-spin text-gray-400" />
      </div>
    )
  }

  const isOnline = driver?.status === 'online' || driver?.status === 'busy'

  const stats = [
    { label: 'الحالة', value: isOnline ? '🟢 متصل' : '🔴 غير متصل', icon: Power },
    { label: 'طلبات اليوم', value: driver?.total_deliveries ?? 0, icon: Package },
    { label: 'التقييم', value: `${(driver?.rating ?? 5).toFixed(1)} ⭐`, icon: Star },
    { label: 'المركبة', value: driver?.vehicle_type ?? '—', icon: Bike },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">أهلاً، مندوب AVEX</h1>
            <p className="text-sm text-gray-500 mt-1">
              {isOnline ? 'أنت متصل وجاهز لاستقبال الطلبات' : 'أنت غير متصل — اضغط للبدء'}
            </p>
          </div>
          <button
            onClick={handleToggleOnline}
            disabled={toggling}
            className="flex items-center gap-2 px-5 h-11 rounded-xl font-medium text-white transition-all active:scale-95 disabled:opacity-50"
            style={{ backgroundColor: isOnline ? '#FF6B35' : '#10B981' }}
          >
            {toggling ? <Loader2 className="w-4 h-4 animate-spin" /> : <Power className="w-4 h-4" />}
            {isOnline ? 'إيقاف' : 'ابدأ العمل'}
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="bg-white rounded-xl p-5 shadow-sm border border-gray-100">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-orange-50 flex items-center justify-center">
                  <Icon className="w-5 h-5 text-orange-500" />
                </div>
                <div>
                  <p className="text-xs text-gray-500">{stat.label}</p>
                  <p className="text-lg font-bold text-gray-900">{stat.value}</p>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-xl p-4 flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-red-800">حدث خطأ</p>
            <p className="text-xs text-red-600 mt-1">{error}</p>
          </div>
        </div>
      )}

      {/* Info */}
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <h2 className="font-bold text-gray-900 mb-3">معلومات المندوب</h2>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between py-2 border-b border-gray-100">
            <span className="text-gray-500">المعرف</span>
            <span className="font-mono text-gray-900" dir="ltr">{driver?.id ?? '—'}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-100">
            <span className="text-gray-500">رقم اللوحة</span>
            <span className="text-gray-900" dir="ltr">{driver?.license_plate ?? '—'}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-100">
            <span className="text-gray-500">عدد الطلبات الكلي</span>
            <span className="text-gray-900">{driver?.total_deliveries ?? 0}</span>
          </div>
          <div className="flex justify-between py-2">
            <span className="text-gray-500">معدل القبول</span>
            <span className="text-gray-900">{driver?.acceptance_rate ?? 100}%</span>
          </div>
        </div>
      </div>

      {/* No driver data */}
      {!driver && !loading && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-4 flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-yellow-800">بيانات المندوب غير متاحة</p>
            <p className="text-xs text-yellow-700 mt-1">
              قد لا يكون لديك سجل في جدول المندوبين بعد. تواصل مع الإدارة لتفعيل حسابك.
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
