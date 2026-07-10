import { useState, useEffect } from 'react'
import { Package, Loader2, Phone, MapPin, Filter } from 'lucide-react'
import { adminAPI } from '@/lib/api'

const statusLabels: Record<string, string> = {
  accepted: 'مقبول', preparing: 'قيد التحضير', ready: 'جاهز', assigned: 'مُسند',
  picked_up: 'تم الاستلام', on_the_way: 'في الطريق', delivering: 'في الطريق', delivered: 'تم التوصيل',
  cancelled: 'ملغي', new: 'جديد',
}
const statuses = ['', 'accepted', 'preparing', 'ready', 'assigned', 'picked_up', 'on_the_way', 'delivered', 'cancelled']

export default function AdminOrdersPage() {
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')

  const load = () => {
    setLoading(true)
    adminAPI.getOrders(filter).then((r) => setOrders(r.orders || [])).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [filter])

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">الطلبات</h1>
      <div className="flex items-center gap-2 mb-4 overflow-x-auto pb-1">
        <Filter className="w-4 h-4 text-gray-400 flex-shrink-0" />
        {statuses.map((s) => (
          <button key={s || 'all'} onClick={() => setFilter(s)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-fluent ${
              filter === s ? 'bg-black text-white' : 'bg-white border border-gray-200 text-gray-600'
            }`}>
            {s ? statusLabels[s] : 'الكل'}
          </button>
        ))}
      </div>

      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       orders.length === 0 ? <div className="py-20 text-center text-gray-400 text-sm">لا توجد طلبات</div> :
       <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
         <div className="overflow-x-auto">
           <table className="w-full text-sm">
             <thead className="bg-gray-50 text-gray-600 text-xs">
               <tr>
                 <th className="px-3 py-2 text-right">رقم الطلب</th>
                 <th className="px-3 py-2 text-right">العميل</th>
                 <th className="px-3 py-2 text-right">المطعم</th>
                 <th className="px-3 py-2 text-right">المندوب</th>
                 <th className="px-3 py-2 text-right">الحالة</th>
                 <th className="px-3 py-2 text-right">الإجمالي</th>
                 <th className="px-3 py-2 text-right">رسم المندوب</th>
                 <th className="px-3 py-2 text-right">هامش المنصة</th>
                 <th className="px-3 py-2 text-right">التاريخ</th>
               </tr>
             </thead>
             <tbody>
               {orders.map((o) => (
                 <tr key={o.id} className="border-t border-gray-100 hover:bg-gray-50">
                   <td className="px-3 py-2 font-mono text-xs" dir="ltr">{o.orderNumber}</td>
                   <td className="px-3 py-2">{o.customerName}</td>
                   <td className="px-3 py-2 text-xs">{o.restaurantName || '-'}</td>
                   <td className="px-3 py-2 text-xs">{o.driverName || '-'} {o.driverTier && `(${o.driverTier})`}</td>
                   <td className="px-3 py-2">
                     <span className={`text-xs px-2 py-0.5 rounded-full ${
                       o.status === 'delivered' ? 'bg-black text-white' :
                       o.status === 'cancelled' ? 'bg-gray-200 text-gray-600' : 'bg-gray-100 text-gray-700'
                     }`}>{statusLabels[o.status] || o.status}</span>
                   </td>
                   <td className="px-3 py-2 font-bold">{(o.total ?? 0).toFixed(2)}</td>
                   <td className="px-3 py-2 text-xs">{(o.driverFee ?? 0).toFixed(2)}</td>
                   <td className="px-3 py-2 text-xs">{(o.platformMargin ?? 0).toFixed(2)}</td>
                   <td className="px-3 py-2 text-xs text-gray-500">{new Date(o.createdAt).toLocaleString('ar-EG', { dateStyle: 'short', timeStyle: 'short' })}</td>
                 </tr>
               ))}
             </tbody>
           </table>
         </div>
       </div>}
    </div>
  )
}
