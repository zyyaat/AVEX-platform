import { useState, useEffect } from 'react'
import { useRouter } from '@/lib/navigation'
import { Loader2, Filter, User, Phone } from 'lucide-react'
import { agentAPI } from '@/lib/api'

const typeLabels: Record<string, string> = { cancellation_request: 'طلب إلغاء', complaint: 'شكوى', other: 'أخرى' }
const filters = [
  { v: '', label: 'الكل' },
  { v: 'mine', label: 'المسندة إليّ' },
  { v: 'unassigned', label: 'غير مسندة' },
  { v: 'open', label: 'مفتوحة' },
]

export default function InboxPage() {
  const router = useRouter()
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')
  const [agentId, setAgentId] = useState('')

  const load = () => {
    setLoading(true)
    agentAPI.getTickets(filter).then((r) => { setTickets(r.tickets || []) })
      .finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [filter])
  useEffect(() => { const id = setInterval(load, 8000); return () => clearInterval(id) }, [filter])

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">صندوق التذاكر</h1>
      <div className="flex items-center gap-2 mb-4 overflow-x-auto pb-1">
        <Filter className="w-4 h-4 text-gray-400 flex-shrink-0" />
        {filters.map((f) => (
          <button key={f.v || 'all'} onClick={() => setFilter(f.v)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-fluent ${
              filter === f.v ? 'bg-black text-white' : 'bg-white border border-gray-200 text-gray-600'
            }`}>{f.label}</button>
        ))}
      </div>

      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       tickets.length === 0 ? <p className="text-center text-gray-400 py-20">لا توجد تذاكر</p> :
       <div className="space-y-2">
         {tickets.map((t) => (
           <button key={t.id} onClick={() => router.push(`/tickets/${t.id}`)}
             className="w-full text-right bg-white rounded-lg border border-gray-200 p-3 hover:border-black transition-fluent">
             <div className="flex items-center justify-between mb-1">
               <div className="flex items-center gap-2">
                 <span className="text-xs font-bold bg-gray-100 px-2 py-0.5 rounded-full">{typeLabels[t.type] || t.type}</span>
                 {t.priority === 'urgent' && <span className="text-[10px] bg-black text-white px-2 py-0.5 rounded-full">عاجل</span>}
                 {t.priority === 'high' && <span className="text-[10px] bg-gray-700 text-white px-2 py-0.5 rounded-full">مرتفع</span>}
               </div>
               <span className={`text-[10px] px-2 py-0.5 rounded-full ${t.status === 'open' ? 'bg-black text-white' : 'bg-gray-100 text-gray-500'}`}>
                 {t.status === 'open' ? 'مفتوحة' : t.status === 'resolved' ? 'تم الحل' : 'مغلقة'}
               </span>
             </div>
             <p className="text-sm text-gray-700 line-clamp-2">{t.reason}</p>
             <div className="flex items-center justify-between mt-2 text-[10px] text-gray-500">
               <span>{t.assignedTo === agentId ? '✓ مسندة إليك' : t.assignedTo ? 'مسندة لموظف آخر' : 'غير مسندة'}</span>
               <span>{new Date(t.createdAt).toLocaleString('ar-EG', { dateStyle: 'short', timeStyle: 'short' })}</span>
             </div>
           </button>
         ))}
       </div>}
    </div>
  )
}
