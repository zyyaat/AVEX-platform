import { useState, useEffect } from 'react'
import { useRouter } from '@/lib/navigation'
import { motion } from 'framer-motion'
import {
  Package, TrendingUp, CheckCircle2, Power, Loader2, Clock, Store, Star, ChevronLeft,
} from 'lucide-react'
import { merchantAPI, type MerchantStats } from '@/lib/api'
import { useAuth } from '@/store/auth'
import { toast } from 'sonner'

export default function MerchantDashboard() {
  const router = useRouter()
  const { merchant, fetchMe } = useAuth()
  const [stats, setStats] = useState<MerchantStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [toggling, setToggling] = useState(false)

  useEffect(() => {
    merchantAPI.getStats().then(setStats).finally(() => setLoading(false))
    const id = setInterval(() => merchantAPI.getStats().then(setStats), 30000)
    return () => clearInterval(id)
  }, [])

  const togglePause = async () => {
    setToggling(true)
    try {
      const next = !merchant?.restaurant?.isActive
      await merchantAPI.togglePause(next)
      await fetchMe()
      toast.success(next ? 'تم فتح المطعم' : 'تم إغلاق المطعم مؤقتاً')
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setToggling(false)
    }
  }

  const isActive = merchant?.restaurant?.isActive

  return (
    <div dir="rtl">
      {/* Restaurant header card */}
      <div className="bg-white rounded-2xl border border-gray-200 p-4 mb-4 shadow-fluent">
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-xl bg-black text-white flex items-center justify-center flex-shrink-0">
            <Store className="w-6 h-6" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-bold text-base truncate">{merchant?.restaurant?.nameAr || 'مطعم'}</p>
            <div className="flex items-center gap-2 mt-0.5">
              <div className="flex items-center gap-0.5">
                <Star className="w-3 h-3 fill-black text-black" />
                <span className="text-xs font-bold">{merchant?.restaurant?.rating?.toFixed(1) ?? "0.0"}</span>
              </div>
              <span className="text-[10px] text-gray-400">({merchant?.restaurant?.ratingCount ?? 0} تقييم)</span>
            </div>
          </div>
          <div className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-bold flex-shrink-0 ${
            isActive ? 'bg-black text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            <div className={`w-2 h-2 rounded-full ${isActive ? 'bg-white animate-pulse' : 'bg-gray-500'}`} />
            {isActive ? 'مفتوح' : 'مغلق'}
          </div>
        </div>
        <button
          onClick={togglePause}
          disabled={toggling}
          className={`w-full mt-3 h-10 rounded-xl text-sm font-bold transition-fluent flex items-center justify-center gap-2 ${
            isActive
              ? 'bg-white border border-gray-300 text-gray-700 hover:bg-gray-50'
              : 'bg-black text-white hover:bg-gray-800'
          } disabled:opacity-50`}
        >
          {toggling ? <Loader2 className="w-4 h-4 animate-spin" /> : <Power className="w-4 h-4" />}
          {isActive ? 'إيقاف مؤقت' : 'فتح المطعم'}
        </button>
      </div>

      {/* Status banner when closed */}
      {!isActive && (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-gray-100 border border-gray-200 rounded-xl p-3 mb-4 flex items-center gap-2"
        >
          <Clock className="w-4 h-4 text-gray-500" />
          <p className="text-xs text-gray-600">مطعمك مغلق حالياً — لن تستقبل طلبات جديدة حتى تفتح</p>
        </motion.div>
      )}

      {/* KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        {loading ? (
          [0, 1, 2, 3].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-4 h-24 animate-pulse">
              <div className="h-3 w-20 bg-gray-100 rounded mb-2" />
              <div className="h-6 w-16 bg-gray-100 rounded" />
            </div>
          ))
        ) : (
          <>
            <KpiCard icon={<Package className="w-4 h-4" />} label="طلبات اليوم" value={stats?.todayCount ?? 0} />
            <KpiCard icon={<Clock className="w-4 h-4" />} label="طلبات نشطة" value={stats?.activeCount ?? 0} highlight />
            <KpiCard icon={<CheckCircle2 className="w-4 h-4" />} label="مكتملة" value={stats?.completedCount ?? 0} />
            <KpiCard icon={<TrendingUp className="w-4 h-4" />} label="إيرادات اليوم" value={`${(stats?.todayRevenue ?? 0).toFixed(0)} ج.م`} />
          </>
        )}
      </div>

      {/* Chart */}
      <div className="bg-white rounded-2xl border border-gray-200 p-4 mb-4 shadow-fluent">
        <h3 className="font-bold text-sm mb-4 flex items-center gap-2">
          <TrendingUp className="w-4 h-4 text-gray-500" />
          إيرادات آخر 7 أيام
        </h3>
        {stats?.daily && stats.daily.length > 0 ? (
          <div className="space-y-2.5">
            {stats.daily.map((d) => {
              const maxCount = Math.max(...stats.daily!.map((x) => x.count), 1)
              const pct = (d.count / maxCount) * 100
              return (
                <div key={d.date} className="flex items-center gap-3">
                  <span className="text-[10px] text-gray-500 w-20">{formatDate(d.date)}</span>
                  <div className="flex-1 bg-gray-100 rounded h-7 overflow-hidden relative">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${pct}%` }}
                      transition={{ duration: 0.5 }}
                      className="absolute inset-y-0 right-0 bg-black rounded flex items-center justify-end px-2"
                    >
                      <span className="text-[10px] text-white font-bold">{d.count}</span>
                    </motion.div>
                  </div>
                  <span className="text-[10px] text-gray-500 w-16 text-left">{(d.revenue ?? 0).toFixed(0)} ج.م</span>
                </div>
              )
            })}
          </div>
        ) : (
          <p className="text-sm text-gray-400 text-center py-8">لا توجد بيانات بعد</p>
        )}
      </div>

      {/* Quick actions */}
      <button
        onClick={() => router.push('/orders')}
        className="w-full h-12 rounded-xl bg-black text-white text-sm font-bold hover:bg-gray-800 transition-fluent flex items-center justify-center gap-2"
      >
        <Package className="w-5 h-5" />
        عرض كل الطلبات
        <ChevronLeft className="w-4 h-4" />
      </button>
    </div>
  )
}

function KpiCard({ icon, label, value, highlight }: { icon: React.ReactNode; label: string; value: React.ReactNode; highlight?: boolean }) {
  return (
    <div className={`rounded-xl border p-4 shadow-fluent ${
      highlight ? 'bg-black text-white border-black' : 'bg-white border-gray-200'
    }`}>
      <div className={`flex items-center justify-between mb-2 ${highlight ? 'text-gray-300' : 'text-gray-500'}`}>
        <span className="text-[10px]">{label}</span>
        <div className={highlight ? 'text-white' : 'text-gray-400'}>{icon}</div>
      </div>
      <p className="text-xl font-bold">{value}</p>
    </div>
  )
}

function formatDate(d: string): string {
  try {
    return new Date(d).toLocaleDateString('ar-EG', { day: '2-digit', month: '2-digit' })
  } catch {
    return d
  }
}
