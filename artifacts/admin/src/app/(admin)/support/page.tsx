import { useState, useEffect } from 'react'
import { LifeBuoy, Loader2, Check, X, MessageSquare, Ban } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

const typeLabels: Record<string, string> = { cancellation_request: 'طلب إلغاء', complaint: 'شكوى', other: 'أخرى' }
const statusLabels: Record<string, string> = { open: 'مفتوحة', resolved: 'تم الحل', rejected: 'مرفوضة' }

export default function AdminSupportPage() {
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<any>(null)
  const [messages, setMessages] = useState<any[]>([])
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)

  const load = () => {
    setLoading(true)
    adminAPI.getTickets().then((r) => setTickets(r.tickets || [])).finally(() => setLoading(false))
  }
  useEffect(() => { load(); const id = setInterval(load, 10000); return () => clearInterval(id) }, [])

  const open = async (t: any) => {
    setSelected(t)
    // fetch ticket detail with messages
    try {
      // Reuse admin endpoint for messages — we have HandleAdminSendMessage but not a GET for ticket.
      // For simplicity, use the agent endpoint via direct fetch with same token
      const token = localStorage.getItem('avex_admin_token')
      const res = await fetch(`/api/v1/support/tickets/${t.id}`, { headers: { Authorization: `Bearer ${token}` } })
      if (res.ok) {
        const d = await res.json()
        setMessages(d.messages || [])
      }
    } catch {}
  }
  useEffect(() => {
    if (selected) open(selected)
  }, [selected])

  const send = async () => {
    if (!body.trim() || !selected) return
    setSending(true)
    try {
      await adminAPI.sendMessage(selected.id, body)
      setBody('')
      open(selected)
    } catch (e: any) { toast.error(e.message) }
    finally { setSending(false) }
  }
  const resolve = async () => {
    if (!selected) return
    try {
      await adminAPI.resolveTicket(selected.id, 'تم الحل من لوحة الإدارة')
      toast.success('تم إغلاق التذكرة')
      load()
      setSelected(null)
    } catch (e: any) { toast.error(e.message) }
  }
  const cancel = async () => {
    if (!selected) return
    if (!confirm('هل تريد إلغاء الطلب المرتبط بهذه التذكرة؟')) return
    try {
      await adminAPI.cancelOrder(selected.id)
      toast.success('تم إلغاء الطلب')
      load()
      setSelected(null)
    } catch (e: any) { toast.error(e.message) }
  }

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">تذاكر الدعم</h1>
      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       <div className="grid md:grid-cols-2 gap-4">
         {/* List */}
         <div className="space-y-2">
           {tickets.length === 0 && <p className="text-center text-gray-400 text-sm py-8">لا توجد تذاكر</p>}
           {tickets.map((t) => (
             <button key={t.id} onClick={() => setSelected(t)}
               className={`w-full text-right bg-white rounded-lg border p-3 transition-fluent ${
                 selected?.id === t.id ? 'border-black' : 'border-gray-200 hover:bg-gray-50'
               }`}>
               <div className="flex items-center justify-between mb-1">
                 <span className="text-xs font-bold bg-gray-100 px-2 py-0.5 rounded-full">{typeLabels[t.type] || t.type}</span>
                 <span className={`text-[10px] px-2 py-0.5 rounded-full ${t.status === 'open' ? 'bg-black text-white' : 'bg-gray-100 text-gray-500'}`}>
                   {statusLabels[t.status] || t.status}
                 </span>
               </div>
               <p className="text-xs text-gray-700 line-clamp-2">{t.reason}</p>
               <p className="text-[10px] text-gray-400 mt-1">{new Date(t.createdAt).toLocaleString('ar-EG', { dateStyle: 'short', timeStyle: 'short' })}</p>
             </button>
           ))}
         </div>
         {/* Detail */}
         <div>
           {!selected ? <p className="text-center text-gray-400 text-sm py-8">اختر تذكرة لعرضها</p> :
            <div className="bg-white rounded-lg border border-gray-200 p-4 sticky top-20">
              <div className="mb-3 pb-3 border-b border-gray-100">
                <p className="font-bold text-sm">{typeLabels[selected.type] || selected.type}</p>
                <p className="text-xs text-gray-500 mt-1">{selected.reason}</p>
                {selected.driverName && <p className="text-xs mt-1">المندوب: {selected.driverName} ({selected.driverPhone})</p>}
                {selected.orderNumber && <p className="text-xs">الطلب: {selected.orderNumber} — {selected.orderStatus}</p>}
              </div>
              <div className="max-h-64 overflow-y-auto space-y-2 mb-3">
                {messages.map((m) => (
                  <div key={m.id} className={`text-sm p-2 rounded-lg ${m.sender === 'agent' || m.sender === 'admin' ? 'bg-black text-white ml-8' : 'bg-gray-100 mr-8'}`}>
                    <p>{m.body}</p>
                    <p className={`text-[9px] mt-1 ${m.sender === 'agent' || m.sender === 'admin' ? 'text-gray-300' : 'text-gray-500'}`}>
                      {m.sender === 'driver' ? 'المندوب' : m.sender === 'agent' ? 'الدعم' : 'الإدارة'} • {new Date(m.createdAt).toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })}
                    </p>
                  </div>
                ))}
                {messages.length === 0 && <p className="text-xs text-gray-400 text-center">لا رسائل</p>}
              </div>
              {selected.status === 'open' && (
                <>
                  <div className="flex gap-2 mb-2">
                    <input value={body} onChange={(e) => setBody(e.target.value)} placeholder="رد..." onKeyDown={(e) => e.key === 'Enter' && send()}
                      className="flex-1 h-10 px-3 rounded-lg border border-gray-200 text-sm focus:outline-none focus:border-black" />
                    <button onClick={send} disabled={sending} className="w-10 h-10 rounded-lg bg-black text-white flex items-center justify-center">
                      <MessageSquare className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={resolve} className="flex-1 h-9 rounded-lg border border-gray-200 text-xs font-medium flex items-center justify-center gap-1">
                      <Check className="w-3.5 h-3.5" /> إغلاق
                    </button>
                    {selected.type === 'cancellation_request' && (
                      <button onClick={cancel} className="flex-1 h-9 rounded-lg bg-black text-white text-xs font-medium flex items-center justify-center gap-1">
                        <Ban className="w-3.5 h-3.5" /> إلغاء الطلب
                      </button>
                    )}
                  </div>
                </>
              )}
            </div>}
         </div>
       </div>}
    </div>
  )
}
