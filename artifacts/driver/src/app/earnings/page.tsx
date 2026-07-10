import { useEffect } from 'react'
import { useRouter } from '@/lib/navigation'
import { TrendingUp, Package, Star } from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { BottomTabBar } from '@/components/BottomTabBar'

export default function EarningsPage() {
  const router = useRouter()
  const { isAuthenticated } = useAuth()
  const { driver, fetchDriver } = useDriver()

  useEffect(() => {
    if (!isAuthenticated) { router.replace('/login'); return }
    fetchDriver()
  }, [isAuthenticated, router, fetchDriver])

  return (
    <div className="min-h-dvh bg-gray-50 pb-16" dir="rtl">
      <div className="bg-white px-5 py-6">
        <h1 className="text-xl font-bold">أرباحي</h1>
      </div>
      <div className="p-4 space-y-4">
        <div className="bg-white rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <TrendingUp className="w-5 h-5 text-gray-500" />
            <span className="text-sm text-gray-500">إجمالي التوصيلات</span>
          </div>
          <p className="text-3xl font-bold">{driver?.total_deliveries || 0}</p>
        </div>
        <div className="bg-white rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <Star className="w-5 h-5 text-gray-500" />
            <span className="text-sm text-gray-500">التقييم</span>
          </div>
          <p className="text-3xl font-bold">{driver?.rating?.toFixed(1) ?? "5.0" || '5.0'} ⭐</p>
        </div>
        <div className="bg-white rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <Package className="w-5 h-5 text-gray-500" />
            <span className="text-sm text-gray-500">معدل القبول</span>
          </div>
          <p className="text-3xl font-bold">{driver?.acceptance_rate || 100}%</p>
        </div>
      </div>
      <BottomTabBar />
    </div>
  )
}
