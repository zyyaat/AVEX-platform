import { useState, useEffect, useRef } from 'react'
import { useParams, useRouter } from '@/lib/navigation'
import { ArrowLeft, Send, User, Phone, Package, Ban, Check, Lock, Loader2, AlertCircle } from 'lucide-react'
import { agentAPI } from '@/lib/api'
import { useAuth } from '@/store/auth'
import { toast } from 'sonner'

const typeLabels: Record<string, string> = { cancellation_request: 'طلب إلغاء', complaint: 'شكوى', other: 'أخرى' }
const priorities = ['low', 'normal', 'high', 'urgent']
const priorityLabels: Record<string, string> = { low: 'منخفضة', normal: 'عادية', high: 'مرتفعة', urgent: 'عاجلة' }

export default function TicketDetailPage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const [ticket, setTicket] = useState<any>(null)
  const [messages, setMessages] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [body, setBody] = useState('')
  const [isInternal, setIsInternal] = useState(false)
  const [sending, setSending] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const endRef = useRef<HTMLDivElement>(null)

  const load = () => {
    agentAPI.getTicket(params.id).then((d) => {
      setTicket(d.ticket)
      setMessages(d.messages || [])
    }).finally(() => setLoading(false))
  }
  useEffect(() => { load(); const id = setInterval(load, 5000); return () => clearInterval(id) }, [params.id])
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages])

  const send = async () => {
    if (!body.trim()) return
    setSending(true)
    try {
      // FIXED: sendMessage now requires agentId parameter (was sending empty string)
      const agent = useAuth.getState().agent
      const agentId = agent?.id || agent?.userId || ''
      await agentAPI.sendMessage(params.id, body, agentId, isInternal)
      setBody('')
      load()
    } catch (e: any) { toast.error(e.message) }
    finally { setSending(false) }
  }
  const assign = async () => { setBusy('assign'); try { await agentAPI.assignTicket(params.id); toast.success('تم الإسناد إليك'); load() } catch (e: any) { toast.error(e.message) } finally { setBusy(null) } }
  const resolve = async () => { setBusy('resolve'); try { await agentAPI.resolveTicket(params.id, 'تم الحل'); toast.success('تم الإغلاق'); load() } catch (e: any) { toast.error(e.message) } finally { setBusy(null) } }
  const cancel = async () => {
    if (!confirm('هل تريد إلغاء الطلب المرتبط؟')) return
    setBusy('cancel'); try { await agentAPI.cancelOrder(params.id); toast.success('تم إلغاء الطلب'); load() } catch (e: any) { toast.error(e.message) } finally { setBusy(null) }
  }
  const setP = async (p: string) => { try { await agentAPI.setPriority(params.id, p); load() } catch (e: any) { toast.error(e.message) } }

  if (loading) return <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
  if (!ticket) return <div className="py-20 text-center text-gray-400">التذكرة غير موجودة</div>

  return (
    <div dir="rtl" className="max-w-3xl mx-auto">
      <header className="sticky top-14 md:top-0 z-20 bg-white border-b border-gray-200 px-4 py-3 -mx-4 md:mx-0 mb-4">
        <div className="flex items-center gap-2 mb-2">
          <button onClick={() => router.back()} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
            <ArrowLeft className="w-5 h-5" />
          </button>
          <span className="text-xs font-bold bg-gray-100 px-2 py-0.5 rounded-full">{typeLabels[ticket.type] || ticket.type}</span>
          <span className={`text-[10px] px-2 py-0.5 rounded-full ${ticket.status === 'open' ? 'bg-black text-white' : 'bg-gray-100 text-gray-500'}`}>
            {ticket.status === 'open' ? 'مفتوحة' : ticket.status === 'resolved' ? 'تم الحل' : 'مغلقة'}
          </span>
        </div>
        <p className="text-sm text-gray-700 mb-2">{ticket.reason}</p>
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <select value={ticket.priority || 'normal'} onChange={(e) => setP(e.target.value)} disabled={ticket.status !== 'open'}
            className="px-2 py-1 rounded border border-gray-200 bg-white">
            {priorities.map((p) => <option key={p} value={p}>{priorityLabels[p]}</option>)}
          </select>
          {ticket.driverName && (
            <button onClick={() => router.push(`/drivers/${ticket.driverId}`)} className="flex items-center gap-1 px-2 py-1 rounded border border-gray-200 hover:bg-gray-50">
              <User className="w-3 h-3" /> {ticket.driverName} <Phone className="w-3 h-3" /> <span dir="ltr">{ticket.driverPhone}</span>
            </button>
          )}
          {ticket.orderNumber && (
            <button onClick={() => router.push(`/orders/${ticket.orderId}`)} className="flex items-center gap-1 px-2 py-1 rounded border border-gray-200 hover:bg-gray-50">
              <Package className="w-3 h-3" /> {ticket.orderNumber}
            </button>
          )}
        </div>
      </header>

      {/* Messages */}
      <div className="space-y-2 mb-4 min-h-[200px]">
        {messages.map((m) => (
          <div key={m.id} className={`flex ${m.sender === 'driver' ? 'justify-start' : 'justify-end'}`}>
            <div className={`max-w-[80%] rounded-2xl px-3 py-2 text-sm ${m.isInternal ? 'bg-yellow-50 border border-yellow-200 text-gray-700' : m.sender === 'driver' ? 'bg-white border border-gray-200' : 'bg-black text-white'}`}>
              {m.isInternal && <div className="flex items-center gap-1 text-[10px] mb-1 opacity-70"><Lock className="w-2.5 h-2.5" /> ملاحظة داخلية</div>}
              <p>{m.body}</p>
              <p className={`text-[9px] mt-1 ${m.sender === 'driver' ? 'text-gray-500' : 'text-gray-300'}`}>
                {m.sender === 'driver' ? 'المندوب' : 'الدعم'} • {new Date(m.createdAt).toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })}
              </p>
            </div>
          </div>
        ))}
        <div ref={endRef} />
      </div>

      {/* Actions */}
      {ticket.status === 'open' && (
        <>
          <div className="flex flex-wrap gap-2 mb-3">
            {!ticket.assignedTo && <button onClick={assign} disabled={busy !== null} className="px-3 h-9 rounded-lg bg-black text-white text-xs font-medium flex items-center gap-1.5"><User className="w-3.5 h-3.5" /> إسناد إليّ</button>}
            <button onClick={resolve} disabled={busy !== null} className="px-3 h-9 rounded-lg border border-gray-200 text-xs font-medium flex items-center gap-1.5"><Check className="w-3.5 h-3.5" /> إغلاق</button>
            {ticket.type === 'cancellation_request' && <button onClick={cancel} disabled={busy !== null} className="px-3 h-9 rounded-lg bg-black text-white text-xs font-medium flex items-center gap-1.5"><Ban className="w-3.5 h-3.5" /> إلغاء الطلب</button>}
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-2 sticky bottom-0">
            <div className="flex items-center gap-2">
              <input value={body} onChange={(e) => setBody(e.target.value)} placeholder="اكتب رداً..." onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), send())}
                className="flex-1 h-10 px-3 rounded-lg border border-gray-200 text-sm focus:outline-none focus:border-black" />
              <button onClick={() => setIsInternal(!isInternal)} title="ملاحظة داخلية"
                className={`w-10 h-10 rounded-lg flex items-center justify-center ${isInternal ? 'bg-yellow-100 border border-yellow-300' : 'border border-gray-200'}`}>
                <Lock className="w-4 h-4" />
              </button>
              <button onClick={send} disabled={sending || !body.trim()} className="w-10 h-10 rounded-lg bg-black text-white flex items-center justify-center disabled:opacity-50">
                {sending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
