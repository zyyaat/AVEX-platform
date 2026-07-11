import { useEffect, useState } from 'react'
import { useRouter } from '@/lib/navigation'
import { LifeBuoy, MessageSquare, Plus } from 'lucide-react'
import { useAuth } from '@/store/auth'
import { supportAPI } from '@/lib/api'
import { BottomTabBar } from '@/components/BottomTabBar'
import { toast } from 'sonner'

export default function SupportPage() {
  const router = useRouter()
  const { isAuthenticated, isInitialized, userID } = useAuth()
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!isInitialized) return; if (!isAuthenticated) { router.replace('/login'); return }
    if (!userID) return
    setLoading(true)
    supportAPI.listMyTickets(userID).then(res => setTickets(res.items || [])).catch(() => {}).finally(() => setLoading(false))
  }, [isInitialized, isAuthenticated, router, userID])

  return (
    <div className="min-h-dvh bg-gray-50 pb-16" dir="rtl">
      <div className="bg-white px-5 py-6 flex items-center justify-between">
        <h1 className="text-xl font-bold">الدعم</h1>
      </div>
      <div className="p-4 space-y-3">
        {loading ? (
          <p className="text-center text-gray-400 py-8">جاري التحميل...</p>
        ) : tickets.length === 0 ? (
          <div className="text-center py-12 text-gray-400">
            <LifeBuoy className="w-12 h-12 mx-auto mb-2 opacity-30" />
            <p>لا توجد تذاكر دعم</p>
          </div>
        ) : (
          tickets.map(t => (
            <div key={t.id} className="bg-white rounded-2xl p-4">
              <p className="font-medium text-sm">{t.subject}</p>
              <p className="text-xs text-gray-500 mt-1">{t.status}</p>
            </div>
          ))
        )}
      </div>
      <BottomTabBar />
    </div>
  )
}
