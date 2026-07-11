import { useEffect, useState } from 'react'
import { useRouter } from '@/lib/navigation'
import { Package, Clock } from 'lucide-react'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { driverAPI } from '@/lib/api'
import { BottomTabBar } from '@/components/BottomTabBar'

export default function HistoryPage() {
  const router = useRouter()
  const { isAuthenticated, isInitialized, userID } = useAuth()
  const { driver, fetchDriver } = useDriver()
  const [orders, setOrders] = useState<any[]>([])

  useEffect(() => {
    if (!isInitialized) return; if (!isAuthenticated) { router.replace('/login'); return }
    fetchDriver()
  }, [isInitialized, isAuthenticated, router, fetchDriver])

  useEffect(() => {
    if (!driver) return
    driverAPI.listDriverOrders(driver.id, 20, 0).then(res => setOrders(res.items || [])).catch(() => {})
  }, [driver])

  return (
    <div className="min-h-dvh bg-gray-50 pb-16" dir="rtl">
      <div className="bg-white px-5 py-6">
        <h1 className="text-xl font-bold">سجلّي</h1>
      </div>
      <div className="p-4 space-y-3">
        {orders.length === 0 ? (
          <div className="text-center py-12 text-gray-400">
            <Package className="w-12 h-12 mx-auto mb-2 opacity-30" />
            <p>لا يوجد طلبات سابقة</p>
          </div>
        ) : (
          orders.map((order) => (
            <div key={order.id} className="bg-white rounded-2xl p-4 flex items-center justify-between">
              <div>
                <p className="font-medium text-sm">#{order.order_number}</p>
                <p className="text-xs text-gray-500">{order.customer_name}</p>
              </div>
              <div className="text-left">
                <p className="font-bold text-sm">{((order.total ?? 0) / 100).toFixed(2)} {order.currency}</p>
                <p className="text-xs text-gray-400">{new Date(order.created_at).toLocaleDateString('ar-EG')}</p>
              </div>
            </div>
          ))
        )}
      </div>
      <BottomTabBar />
    </div>
  )
}
