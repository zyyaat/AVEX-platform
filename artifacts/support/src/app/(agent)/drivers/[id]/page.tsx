import { useState, useEffect } from 'react'
import { useParams, useRouter } from '@/lib/navigation'
import { ArrowLeft, Phone, Award, Bike, Package, Loader2, Star, TrendingUp } from 'lucide-react'
import { agentAPI } from '@/lib/api'

const statusLabels: Record<string, string> = {
  delivered: 'تم التوصيل', cancelled: 'ملغي', on_the_way: 'في الطريق', picked_up: 'تم الاستلام', assigned: 'مُسند',
}

export default function AgentDriverPage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    agentAPI.getDriver(params.id).then(setData).finally(() => setLoading(false))
  }, [params.id])

  if (loading) return <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
  if (!data || !data.driver) return <div className="py-20 text-center text-gray-400">المندوب غير موجود</div>

  const d = data.driver, s = data.stats || {}
  return (
    <div dir="rtl" className="max-w-2xl mx-auto">
      <button onClick={() => router.back()} className="mb-3 flex items-center gap-1 text-sm text-gray-600 hover:text-black">
        <ArrowLeft className="w-4 h-4" /> رجوع
      </button>

      <div className="bg-white rounded-lg border border-gray-200 p-4 mb-3">
        <div className="flex items-center gap-3 mb-3">
          <div className="w-14 h-14 rounded-full flex items-center justify-center text-white text-xl font-bold" style={{ backgroundColor: d.tierColor || '#000' }}>
            {d.name?.charAt(0)}
          </div>
          <div className="flex-1">
            <p className="font-bold text-lg">{d.name}</p>
            <p className="text-xs text-gray-500" dir="ltr">{d.phone}</p>
          </div>
          <div className="text-left">
            <span className="text-xs px-2 py-0.5 rounded-full text-white" style={{ backgroundColor: d.tierColor || '#000' }}>{d.tierName}</span>
            <p className="text-xs text-gray-500 mt-1">{d.isOnline ? '● متصل' : '○ غير متصل'}</p>
          </div>
        </div>
        <a href={`tel:${d.phone}`} className="w-full h-10 rounded-lg bg-black text-white flex items-center justify-center gap-2 text-sm font-medium">
          <Phone className="w-4 h-4" /> اتصال بالمندوب
        </a>
      </div>

      <div className="grid grid-cols-3 gap-2 mb-3">
        <div className="bg-white rounded-lg border border-gray-200 p-3 text-center">
          <Star className="w-4 h-4 mx-auto mb-1 text-gray-500" />
          <p className="text-base font-bold">{(s.rating ?? 0).toFixed(1)}</p>
          <p className="text-[10px] text-gray-400">تقييم ({s.ratingCount})</p>
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-3 text-center">
          <Award className="w-4 h-4 mx-auto mb-1 text-gray-500" />
          <p className="text-base font-bold">{(s.acceptanceRate ?? 0).toFixed(0)}%</p>
          <p className="text-[10px] text-gray-400">قبول</p>
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-3 text-center">
          <TrendingUp className="w-4 h-4 mx-auto mb-1 text-gray-500" />
          <p className="text-base font-bold">{(s.totalEarnings ?? 0).toFixed(0)}</p>
          <p className="text-[10px] text-gray-400">أرباح</p>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-3 mb-3">
        <p className="text-xs text-gray-500 mb-2">إحصائيات تفصيلية</p>
        <div className="grid grid-cols-2 gap-2 text-xs">
          <div>طلبات مقبولة: <b>{s.acceptedOrders}</b></div>
          <div>طلبات مرفوضة: <b>{s.rejectedOrders}</b></div>
          <div>طلبات مكتملة: <b>{s.completedOrders}</b></div>
          <div>الالتزام بالوقت: <b>{(s.onTimeRate ?? 0).toFixed(0)}%</b></div>
          <div>نسبة الإكمال: <b>{(s.completionRate ?? 0).toFixed(0)}%</b></div>
        </div>
      </div>

      {data.recentOrders && data.recentOrders.length > 0 && (
        <div className="bg-white rounded-lg border border-gray-200 p-3">
          <p className="text-xs text-gray-500 mb-2">آخر الطلبات</p>
          <div className="space-y-1 text-xs">
            {data.recentOrders.map((o: any) => (
              <div key={o.id} className="flex items-center justify-between py-1.5 border-b border-gray-100 last:border-0">
                <span dir="ltr">{o.orderNumber}</span>
                <span className="text-gray-500">{statusLabels[o.status] || o.status}</span>
                <span className="font-bold">{(o.earnings ?? 0).toFixed(2)} ج.م</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
