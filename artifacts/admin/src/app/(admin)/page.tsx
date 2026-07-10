import { useState, useEffect } from 'react'
import { Package, Bike, LifeBuoy, Users, DollarSign, TrendingUp, Store, Loader2, AlertCircle } from 'lucide-react'
import { adminAPI } from '@/lib/api'

export default function AdminDashboard() {
  const [stats, setStats] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    adminAPI.getDashboard().then(setStats).finally(() => setLoading(false))
    const id = setInterval(() => adminAPI.getDashboard().then(setStats), 30000)
    return () => clearInterval(id)
  }, [])

  if (loading || !stats) return <div className="flex items-center justify-center py-20"><Loader2 className="w-6 h-6 animate-spin" /></div>

  const kpis = [
    { label: 'طلبات اليوم', value: stats.todayOrders, icon: Package },
    { label: 'طلبات نشطة', value: stats.activeOrders, icon: AlertCircle },
    { label: 'مندوبين نشطين', value: stats.onlineDrivers, icon: Bike },
    { label: 'إيرادات اليوم', value: `${(stats.todayRevenue ?? 0).toFixed(2)} ج.م`, icon: DollarSign },
    { label: 'هامش المنصة', value: `${(stats.platformMargin ?? 0).toFixed(2)} ج.م`, icon: TrendingUp },
    { label: 'تذاكر مفتوحة', value: stats.openTickets, icon: LifeBuoy },
    { label: 'عملاء', value: stats.totalCustomers, icon: Users },
    { label: 'مطاعم', value: stats.totalRestaurants, icon: Store },
  ]

  const statusLabels: Record<string, string> = {
    accepted: 'مقبول', preparing: 'قيد التحضير', ready: 'جاهز', assigned: 'مُسند',
    picked_up: 'تم الاستلام', on_the_way: 'في الطريق', delivering: 'في الطريق', delivered: 'تم التوصيل',
    cancelled: 'ملغي', new: 'جديد',
  }

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">لوحة المعلومات</h1>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {kpis.map((k) => {
          const Icon = k.icon
          return (
            <div key={k.label} className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-gray-500">{k.label}</span>
                <Icon className="w-4 h-4 text-gray-400" />
              </div>
              <p className="text-xl font-bold">{k.value}</p>
            </div>
          )
        })}
      </div>

      <div className="grid md:grid-cols-2 gap-4">
        {/* Daily revenue */}
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="font-bold text-sm mb-3">آخر 7 أيام — الطلبات والإيرادات</h3>
          {stats.daily && stats.daily.length > 0 ? (
            <div className="space-y-2">
              {stats.daily.map((d: any) => (
                <div key={d.date} className="flex items-center gap-3">
                  <span className="text-xs text-gray-500 w-20">{d.date}</span>
                  <div className="flex-1 bg-gray-100 rounded h-6 overflow-hidden relative">
                    <div className="absolute inset-y-0 right-0 bg-black" style={{ width: `${Math.min(100, d.count * 10)}%` }} />
                  </div>
                  <span className="text-xs font-bold w-12 text-left">{d.count}</span>
                  <span className="text-xs text-gray-500 w-20 text-left">{d.revenue.toFixed(0)} ج.م</span>
                </div>
              ))}
            </div>
          ) : <p className="text-sm text-gray-400 text-center py-6">لا توجد بيانات</p>}
        </div>

        {/* Status breakdown */}
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="font-bold text-sm mb-3">طلبات اليوم حسب الحالة</h3>
          {Object.keys(stats.byStatus || {}).length > 0 ? (
            <div className="space-y-2">
              {Object.entries(stats.byStatus).map(([st, count]: any) => (
                <div key={st} className="flex items-center justify-between text-sm py-1.5 border-b border-gray-100 last:border-0">
                  <span className="text-gray-700">{statusLabels[st] || st}</span>
                  <span className="font-bold">{count}</span>
                </div>
              ))}
            </div>
          ) : <p className="text-sm text-gray-400 text-center py-6">لا توجد طلبات اليوم</p>}
        </div>
      </div>
    </div>
  )
}
