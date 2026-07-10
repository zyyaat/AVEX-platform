import { useState, useEffect } from 'react'
import { useParams, useRouter } from '@/lib/navigation'
import { ArrowLeft, Phone, MapPin, User, Loader2, Package } from 'lucide-react'
import { agentAPI } from '@/lib/api'

const statusLabels: Record<string, string> = {
  accepted: 'مقبول', preparing: 'قيد التحضير', ready: 'جاهز', assigned: 'مُسند',
  picked_up: 'تم الاستلام', on_the_way: 'في الطريق', delivering: 'في الطريق', delivered: 'تم التوصيل', cancelled: 'ملغي',
}

export default function AgentOrderPage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const [order, setOrder] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    agentAPI.getOrder(params.id).then((d) => setOrder(d.order)).finally(() => setLoading(false))
  }, [params.id])

  if (loading) return <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
  if (!order) return <div className="py-20 text-center text-gray-400">الطلب غير موجود</div>

  return (
    <div dir="rtl" className="max-w-2xl mx-auto">
      <button onClick={() => router.back()} className="mb-3 flex items-center gap-1 text-sm text-gray-600 hover:text-black">
        <ArrowLeft className="w-4 h-4" /> رجوع
      </button>

      <div className="bg-white rounded-lg border border-gray-200 p-4 mb-3">
        <div className="flex items-center justify-between mb-2">
          <p className="font-bold" dir="ltr">{order.orderNumber}</p>
          <span className={`text-xs px-2 py-0.5 rounded-full ${order.status === 'delivered' ? 'bg-black text-white' : order.status === 'cancelled' ? 'bg-gray-200 text-gray-600' : 'bg-gray-100 text-gray-700'}`}>
            {statusLabels[order.status] || order.status}
          </span>
        </div>
        <div className="space-y-1 text-sm text-gray-700">
          <div className="flex items-center gap-2"><User className="w-4 h-4 text-gray-400" /> {order.customerName} <a href={`tel:${order.phone}`} className="text-xs text-gray-500 mr-auto" dir="ltr">{order.phone}</a></div>
          <div className="flex items-start gap-2"><MapPin className="w-4 h-4 text-gray-400 mt-0.5" /> <span className="flex-1 text-xs">{order.locationAddress}</span></div>
          <a href={order.locationUrl} target="_blank" className="text-xs underline">فتح الخريطة</a>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-4 mb-3">
        <p className="font-bold text-sm mb-2">الأصناف</p>
        <div className="space-y-1 text-sm">
          {(order.items || []).map((it: any, i: number) => (
            <div key={i} className="flex justify-between"><span>{it.quantity}× {it.name}</span><span className="text-gray-500">{(it.price * it.quantity).toFixed(2)} ج.م</span></div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="bg-white rounded-lg border border-gray-200 p-3">
          <p className="text-xs text-gray-500 mb-1">المطعم</p>
          <p className="font-bold text-sm">{order.restaurantNameAr || order.restaurantName}</p>
        </div>
        <button onClick={() => order.driverId && router.push(`/drivers/${order.driverId}`)} className="bg-white rounded-lg border border-gray-200 p-3 text-right hover:border-black">
          <p className="text-xs text-gray-500 mb-1">المندوب</p>
          <p className="font-bold text-sm">{order.driverName || '-'}</p>
          {order.driverPhone && <p className="text-xs text-gray-500" dir="ltr">{order.driverPhone}</p>}
        </button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-3 mt-3">
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div>الإجمالي: <b>{(order.total ?? 0).toFixed(2)} ج.م</b></div>
          <div>المجموع الفرعي: {(order.subtotal ?? 0).toFixed(2)}</div>
          <div>رسوم التوصيل: {(order.deliveryFee ?? 0).toFixed(2)}</div>
          <div>رسم المندوب: {(order.driverFee ?? 0).toFixed(2)}</div>
          {order.discount > 0 && <div>الخصم: -{(order.discount ?? 0).toFixed(2)}</div>}
          <div>طريقة الدفع: {order.paymentMethod === 'cash' ? 'نقدي' : order.paymentMethod}</div>
        </div>
      </div>
    </div>
  )
}
