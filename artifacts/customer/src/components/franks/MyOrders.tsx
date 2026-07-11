
import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { Package, Clock, CheckCircle2, ChefHat, Bike, Home, X, MapPin, ExternalLink, ArrowRight, ShoppingBag, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ordersAPI } from '@/lib/api'
import { useAuth } from '@/store/auth'

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: typeof Package }> = {
  new: { label: 'جديد', color: 'bg-gray-100 text-gray-700', icon: Clock },
  accepted: { label: 'مقبول', color: 'bg-gray-100 text-gray-700', icon: CheckCircle2 },
  preparing: { label: 'قيد التحضير', color: 'bg-gray-100 text-gray-700', icon: ChefHat },
  ready: { label: 'جاهز', color: 'bg-cyan-100 text-cyan-700', icon: Package },
  picked_up: { label: 'تم الاستلام', color: 'bg-orange-100 text-orange-700', icon: Bike },
  delivering: { label: 'في الطريق', color: 'bg-indigo-100 text-indigo-700', icon: Bike },
  delivered: { label: 'تم التوصيل', color: 'bg-gray-100 text-gray-700', icon: Home },
  cancelled: { label: 'ملغى', color: 'bg-red-100 text-red-700', icon: X },
  rejected: { label: 'مرفوض', color: 'bg-red-100 text-red-700', icon: X },
}

interface MyOrdersProps {
  onBack: () => void
  onLoginRequired: () => void
}

export function MyOrders({ onBack, onLoginRequired }: MyOrdersProps) {
  const { isAuthenticated, user } = useAuth()
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated) { onLoginRequired(); return }
    fetchOrders()
    const interval = setInterval(fetchOrders, 15000)
    return () => clearInterval(interval)
  }, [isAuthenticated])

  const fetchOrders = async () => {
    try {
      const data = await ordersAPI.getMyOrders()
      // FIXED: response shape is { items, total }, not { orders }
      setOrders(data.items || [])
    } catch {} finally { setLoading(false) }
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-dvh bg-gray-50 flex items-center justify-center p-4">
        <div className="text-center bg-white rounded-lg border border-gray-200 p-8 max-w-sm">
          <Package className="w-12 h-12 text-black mx-auto mb-3" />
          <h2 className="text-xl font-bold mb-2">سجّل دخولك أولاً</h2>
          <Button onClick={onBack} className="rounded-xl bg-black">العودة للمتجر</Button>
        </div>
      </div>
    )
  }

  const formatDate = (d: string) => new Date(d).toLocaleString('ar', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })

  return (
    <div className="min-h-dvh bg-gray-50" dir="rtl">
      <header className="sticky top-0 z-30 bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 h-14 flex items-center justify-between">
          <button onClick={onBack} className="flex items-center gap-2 text-sm font-medium text-gray-700"><ArrowRight className="w-4 h-4" /> العودة للمتجر</button>
          <h1 className="font-bold text-base text-gray-900">طلباتي</h1>
        </div>
      </header>
      <div className="container mx-auto px-4 py-6 max-w-2xl space-y-4">
        <div className="bg-white rounded-2xl border p-4 flex items-center gap-3">
          <div className="w-12 h-12 rounded-full bg-gray-50 flex items-center justify-center"><ShoppingBag className="w-6 h-6 text-black" /></div>
          <div className="flex-1">
            <p className="font-bold text-gray-900">مرحباً، {user?.name}!</p>
            <p className="text-xs text-gray-500">{orders.length > 0 ? `${orders.length} طلب` : 'لا توجد طلبات'}{user?.loyaltyPoints ? ` • ${user.loyaltyPoints} نقطة` : ''}</p>
          </div>
        </div>
        {loading ? <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-black" /></div>
        : orders.length === 0 ? <div className="text-center py-20 bg-white rounded-2xl border"><Package className="w-12 h-12 text-gray-300 mx-auto mb-3" /><h3 className="text-lg font-bold mb-1">لا توجد طلبات</h3><Button onClick={onBack} className="rounded-xl bg-black">تصفّح القائمة</Button></div>
        : orders.map((order, idx) => {
          const cfg = STATUS_CONFIG[order.status] || STATUS_CONFIG.new
          const Icon = cfg.icon
          return (
            <motion.div key={order.id} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: idx * 0.05 }} className="bg-white rounded-2xl border p-4">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${cfg.color}`}><Icon className="w-5 h-5" /></div>
                  <div><p className="font-mono text-sm" dir="ltr">{order.orderNumber}</p><p className="text-xs text-gray-500">{formatDate(order.createdAt)}</p></div>
                </div>
                <span className={`text-xs font-bold px-3 py-1 rounded-full ${cfg.color}`}>{cfg.label}</span>
              </div>
              <div className="bg-gray-50 rounded-xl p-3 mb-3 space-y-1.5">
                {order.items?.map((item: any) => <div key={item.id} className="flex items-center justify-between text-sm"><span className="flex items-center gap-2"><span className="bg-white rounded px-1.5 py-0.5 text-xs font-bold border">{item.quantity}×</span>{item.name}</span><span className="font-bold">{(item.price * item.quantity).toFixed(2)} ج.م</span></div>)}
              </div>
              <div className="space-y-1 text-sm">
                <div className="flex justify-between text-gray-500"><span>المجموع الفرعي</span><span>{order.subtotal?.toFixed(2)} ج.م</span></div>
                <div className="flex justify-between text-gray-500"><span>رسوم التوصيل</span><span>{order.deliveryFee === 0 ? <span className="text-gray-500 font-bold">مجاني</span> : `${order.deliveryFee?.toFixed(2)} ج.م`}</span></div>
                {order.discount > 0 && <div className="flex justify-between text-gray-500"><span>الخصم</span><span className="font-bold">-{order.discount?.toFixed(2)} ج.م</span></div>}
                <div className="flex justify-between font-bold text-base pt-1 border-t"><span>الإجمالي</span><span className="text-black">{order.total?.toFixed(2)} ج.م</span></div>
              </div>
              {order.locationUrl && <a href={order.locationUrl} target="_blank" rel="noopener noreferrer" className="mt-3 flex items-center justify-between gap-2 bg-gray-50 border border-gray-200 rounded-xl p-2.5"><span className="text-xs font-medium text-black flex items-center gap-2"><MapPin className="w-4 h-4" /> موقع التوصيل</span><ExternalLink className="w-4 h-4 text-gray-500" /></a>}
            </motion.div>
          )
        })}
      </div>
    </div>
  )
}
