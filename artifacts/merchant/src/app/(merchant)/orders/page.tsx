import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Package, Loader2, Phone, MapPin, Filter, ChefHat, CheckCircle2, X, Clock, User, Navigation,
} from 'lucide-react'
import { merchantAPI, type MerchantOrder, type OrderItem } from '@/lib/api'
import { toast } from 'sonner'

const statusLabels: Record<string, string> = {
  accepted: 'جديد',
  preparing: 'قيد التحضير',
  ready: 'جاهز',
  assigned: 'بانتظار المندوب',
  picked_up: 'خرج مع المندوب',
  on_the_way: 'في الطريق',
  delivering: 'في الطريق',
  delivered: 'تم التوصيل',
  cancelled: 'ملغي',
}

const statusColors: Record<string, string> = {
  accepted: 'bg-black text-white',
  preparing: 'bg-gray-700 text-white',
  ready: 'bg-gray-500 text-white',
  assigned: 'bg-gray-300 text-gray-700',
  picked_up: 'bg-gray-200 text-gray-700',
  on_the_way: 'bg-gray-200 text-gray-700',
  delivering: 'bg-gray-200 text-gray-700',
  delivered: 'bg-gray-100 text-gray-500',
  cancelled: 'bg-gray-100 text-gray-400',
}

const filters = [
  { v: '', label: 'الكل' },
  { v: 'accepted', label: 'جديد' },
  { v: 'preparing', label: 'قيد التحضير' },
  { v: 'ready', label: 'جاهز' },
  { v: 'picked_up', label: 'مع المندوب' },
  { v: 'delivered', label: 'تم التوصيل' },
  { v: 'cancelled', label: 'ملغي' },
]

export default function MerchantOrdersPage() {
  const [orders, setOrders] = useState<MerchantOrder[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')
  const [selected, setSelected] = useState<MerchantOrder | null>(null)
  const [items, setItems] = useState<OrderItem[]>([])

  const load = () => {
    setLoading(true)
    merchantAPI.getOrders(filter).then((r) => setOrders(r.orders || [])).finally(() => setLoading(false))
  }
  useEffect(() => {
    load()
    const id = setInterval(load, 5000)
    return () => clearInterval(id)
  }, [filter])

  const open = async (o: MerchantOrder) => {
    setSelected(o)
    setItems([])
    try {
      const r = await merchantAPI.getOrderItems(o.id)
      setItems(r.items || [])
    } catch {}
  }

  const updateStatus = async (id: string, status: string) => {
    try {
      await merchantAPI.updateOrderStatus(id, status)
      toast.success(status === 'preparing' ? 'بدأ التحضير' : status === 'ready' ? 'الطلب جاهز' : 'تم التحديث')
      load()
      if (selected?.id === id) setSelected(null)
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const newCount = orders.filter((o) => o.status === 'accepted').length

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">الطلبات</h1>
        {newCount > 0 && (
          <span className="bg-black text-white text-xs font-bold px-2.5 py-1 rounded-full">
            {newCount} جديد
          </span>
        )}
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2 mb-4 overflow-x-auto pb-1">
        <Filter className="w-4 h-4 text-gray-400 flex-shrink-0" />
        {filters.map((f) => (
          <button
            key={f.v || 'all'}
            onClick={() => setFilter(f.v)}
            className={`px-3 py-1.5 rounded-full text-xs font-bold whitespace-nowrap transition-fluent ${
              filter === f.v ? 'bg-black text-white' : 'bg-white border border-gray-200 text-gray-600'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Orders grid (KDS-style) */}
      {loading ? (
        <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
      ) : orders.length === 0 ? (
        <div className="text-center py-20">
          <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mx-auto mb-4">
            <Package className="w-7 h-7 text-gray-300" />
          </div>
          <p className="text-sm text-gray-500 font-medium">لا توجد طلبات</p>
          <p className="text-xs text-gray-400 mt-1">ستظهر الطلبات الجديدة هنا فور استلامها</p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-3">
          {orders.map((o, idx) => (
            <motion.div
              key={o.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: idx * 0.04 }}
              className={`bg-white rounded-xl border-2 p-4 shadow-fluent ${
                o.status === 'accepted' ? 'border-black' : 'border-gray-200'
              }`}
            >
              {/* Header */}
              <div className="flex items-start justify-between mb-2">
                <div>
                  <p className="font-bold text-sm" dir="ltr">{o.orderNumber}</p>
                  <p className="text-[10px] text-gray-500 flex items-center gap-1 mt-0.5">
                    <Clock className="w-2.5 h-2.5" />
                    {new Date(o.createdAt).toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })}
                  </p>
                </div>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${statusColors[o.status] || 'bg-gray-100 text-gray-500'}`}>
                  {statusLabels[o.status] || o.status}
                </span>
              </div>

              {/* Customer */}
              <div className="bg-gray-50 rounded-lg p-2 mb-2">
                <p className="font-bold text-sm flex items-center gap-1.5">
                  <User className="w-3 h-3 text-gray-400" />
                  {o.customerName}
                </p>
                <p className="text-[10px] text-gray-500 mt-0.5" dir="ltr">{o.phone}</p>
              </div>

              {/* Items summary */}
              <p className="text-xs text-gray-600 line-clamp-2 mb-2">{o.itemsSummary}</p>
              <p className="text-xs text-gray-500 mb-3">
                {o.itemsCount} أصناف • <b className="text-black">{(o.total ?? 0).toFixed(2)} ج.م</b>
              </p>

              {/* Actions */}
              <div className="flex gap-1.5">
                <button
                  onClick={() => open(o)}
                  className="flex-1 h-9 rounded-lg border border-gray-200 text-xs font-bold hover:bg-gray-50 transition-fluent"
                >
                  تفاصيل
                </button>
                {o.status === 'accepted' && (
                  <button
                    onClick={() => updateStatus(o.id, 'preparing')}
                    className="flex-1 h-9 rounded-lg bg-black text-white text-xs font-bold flex items-center justify-center gap-1 hover:bg-gray-800 transition-fluent"
                  >
                    <ChefHat className="w-3 h-3" />
                    تحضير
                  </button>
                )}
                {o.status === 'preparing' && (
                  <button
                    onClick={() => updateStatus(o.id, 'ready')}
                    className="flex-1 h-9 rounded-lg bg-black text-white text-xs font-bold flex items-center justify-center gap-1 hover:bg-gray-800 transition-fluent"
                  >
                    <CheckCircle2 className="w-3 h-3" />
                    جاهز
                  </button>
                )}
              </div>
            </motion.div>
          ))}
        </div>
      )}

      {/* Detail modal */}
      <AnimatePresence>
        {selected && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/60 flex items-end sm:items-center justify-center p-0 sm:p-4"
            onClick={(e) => e.target === e.currentTarget && setSelected(null)}
          >
            <motion.div
              initial={{ y: '100%', opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: '100%', opacity: 0 }}
              transition={{ type: 'spring', damping: 28, stiffness: 320 }}
              className="bg-white w-full sm:max-w-md sm:rounded-2xl rounded-t-2xl max-h-[90vh] overflow-y-auto"
              dir="rtl"
            >
              {/* Header */}
              <div className="bg-black text-white px-5 py-4 flex items-center justify-between sticky top-0">
                <div>
                  <p className="text-xs text-gray-300">طلب</p>
                  <p className="font-bold text-lg" dir="ltr">{selected.orderNumber}</p>
                </div>
                <button
                  onClick={() => setSelected(null)}
                  className="w-8 h-8 rounded-full hover:bg-white/10 flex items-center justify-center"
                  aria-label="إغلاق"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="p-5 space-y-4">
                {/* Status badge */}
                <div className="flex items-center justify-between">
                  <span className={`text-xs px-3 py-1 rounded-full font-bold ${statusColors[selected.status] || 'bg-gray-100'}`}>
                    {statusLabels[selected.status] || selected.status}
                  </span>
                  <span className="text-xs text-gray-500">
                    {new Date(selected.createdAt).toLocaleString('ar-EG', { dateStyle: 'short', timeStyle: 'short' })}
                  </span>
                </div>

                {/* Customer info */}
                <div className="bg-gray-50 rounded-xl p-3 border border-gray-200 space-y-2">
                  <div className="flex items-center gap-2">
                    <User className="w-4 h-4 text-gray-500" />
                    <span className="font-bold text-sm">{selected.customerName}</span>
                  </div>
                  <a href={`tel:${selected.phone}`} className="flex items-center gap-2 text-sm">
                    <Phone className="w-4 h-4 text-gray-500" />
                    <span dir="ltr" className="font-bold">{selected.phone}</span>
                  </a>
                  <div className="flex items-start gap-2 text-xs">
                    <MapPin className="w-4 h-4 text-gray-400 mt-0.5" />
                    <span className="text-gray-600 flex-1">{selected.locationAddress}</span>
                  </div>
                  <a
                    href={selected.locationUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1.5 text-xs text-black underline"
                  >
                    <Navigation className="w-3 h-3" />
                    فتح الخريطة
                  </a>
                </div>

                {/* Items */}
                <div className="bg-gray-50 rounded-xl p-3 border border-gray-200">
                  <p className="text-xs font-bold mb-2 flex items-center gap-1">
                    <Package className="w-3.5 h-3.5" />
                    الأصناف ({items.length})
                  </p>
                  <div className="space-y-1.5">
                    {items.length === 0 ? (
                      <p className="text-xs text-gray-400 text-center py-2">جاري التحميل...</p>
                    ) : (
                      items.map((it) => (
                        <div key={it.id} className="flex items-center justify-between text-sm">
                          <span className="flex items-center gap-2">
                            <span className="bg-black text-white w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold">
                              {it.quantity}
                            </span>
                            {it.name}
                          </span>
                          <span className="text-gray-500 text-xs">{(it.price * it.quantity).toFixed(2)} ج.م</span>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Totals */}
                <div className="bg-black text-white rounded-xl p-4">
                  <div className="flex items-center justify-between text-xs text-gray-300 mb-1">
                    <span>المجموع الفرعي</span>
                    <span>{(selected.subtotal ?? 0).toFixed(2)} ج.م</span>
                  </div>
                  <div className="flex items-center justify-between text-xs text-gray-300 mb-1">
                    <span>رسوم التوصيل</span>
                    <span>{(selected.deliveryFee ?? 0).toFixed(2)} ج.م</span>
                  </div>
                  {selected.discount > 0 && (
                    <div className="flex items-center justify-between text-xs text-gray-300 mb-1">
                      <span>الخصم</span>
                      <span>-{(selected.discount ?? 0).toFixed(2)} ج.م</span>
                    </div>
                  )}
                  <div className="border-t border-white/20 mt-2 pt-2 flex items-center justify-between font-bold">
                    <span>الإجمالي</span>
                    <span className="text-lg">{(selected.total ?? 0).toFixed(2)} ج.م</span>
                  </div>
                  <div className="text-[10px] text-gray-300 mt-1">طريقة الدفع: {selected.paymentMethod === 'cash' ? 'نقدي' : selected.paymentMethod}</div>
                </div>

                {/* Action buttons */}
                {selected.status === 'accepted' && (
                  <button
                    onClick={() => updateStatus(selected.id, 'preparing')}
                    className="w-full h-12 rounded-xl bg-black text-white font-bold flex items-center justify-center gap-2 hover:bg-gray-800 transition-fluent"
                  >
                    <ChefHat className="w-5 h-5" />
                    بدء التحضير
                  </button>
                )}
                {selected.status === 'preparing' && (
                  <button
                    onClick={() => updateStatus(selected.id, 'ready')}
                    className="w-full h-12 rounded-xl bg-black text-white font-bold flex items-center justify-center gap-2 hover:bg-gray-800 transition-fluent"
                  >
                    <CheckCircle2 className="w-5 h-5" />
                    جاهز للاستلام
                  </button>
                )}
                {(selected.status === 'picked_up' || selected.status === 'on_the_way' || selected.status === 'delivering') && (
                  <div className="bg-gray-100 rounded-xl p-3 text-center text-sm text-gray-600">
                    الطلب خرج مع المندوب — في انتظار التسليم
                  </div>
                )}
                {selected.status === 'delivered' && (
                  <div className="bg-gray-100 rounded-xl p-3 text-center text-sm text-gray-600">
                    تم التوصيل بنجاح ✓
                  </div>
                )}
                {selected.status === 'cancelled' && (
                  <div className="bg-gray-100 rounded-xl p-3 text-center text-sm text-gray-600">
                    تم إلغاء الطلب
                  </div>
                )}
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
