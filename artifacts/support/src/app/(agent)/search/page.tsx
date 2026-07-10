import { useState } from 'react'
import { useRouter } from '@/lib/navigation'
import { Search, User, Bike, Package, Loader2 } from 'lucide-react'
import { agentAPI } from '@/lib/api'

export default function SearchPage() {
  const router = useRouter()
  const [q, setQ] = useState('')
  const [results, setResults] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const search = async (query: string) => {
    setQ(query)
    if (query.length < 3) { setResults(null); return }
    setLoading(true)
    try { setResults(await agentAPI.search(query)) } finally { setLoading(false) }
  }

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">بحث شامل</h1>
      <div className="relative mb-6">
        <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
        <input autoFocus value={q} onChange={(e) => search(e.target.value)} placeholder="ابحث بالاسم، الهاتف، رقم الطلب..."
          className="w-full h-12 pr-11 pl-4 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
      </div>

      {loading && <div className="text-center py-8"><Loader2 className="w-5 h-5 animate-spin mx-auto" /></div>}

      {results && !loading && (
        <div className="space-y-4">
          {results.customers.length > 0 && (
            <div>
              <h3 className="text-sm font-bold mb-2 flex items-center gap-2"><User className="w-4 h-4" /> العملاء</h3>
              <div className="space-y-1">
                {results.customers.map((c: any) => (
                  <div key={c.id} className="bg-white rounded-lg border border-gray-200 p-3 text-sm">
                    <p className="font-bold">{c.name}</p>
                    <p className="text-xs text-gray-500" dir="ltr">{c.phone}</p>
                  </div>
                ))}
              </div>
            </div>
          )}
          {results.drivers.length > 0 && (
            <div>
              <h3 className="text-sm font-bold mb-2 flex items-center gap-2"><Bike className="w-4 h-4" /> المندوبون</h3>
              <div className="space-y-1">
                {results.drivers.map((d: any) => (
                  <button key={d.id} onClick={() => router.push(`/drivers/${d.id}`)}
                    className="w-full text-right bg-white rounded-lg border border-gray-200 p-3 text-sm hover:border-black">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-bold">{d.name}</p>
                        <p className="text-xs text-gray-500" dir="ltr">{d.phone}</p>
                      </div>
                      <span className="text-xs bg-gray-100 px-2 py-0.5 rounded-full">{d.tierName || '-'}</span>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}
          {results.orders.length > 0 && (
            <div>
              <h3 className="text-sm font-bold mb-2 flex items-center gap-2"><Package className="w-4 h-4" /> الطلبات</h3>
              <div className="space-y-1">
                {results.orders.map((o: any) => (
                  <button key={o.id} onClick={() => router.push(`/orders/${o.id}`)}
                    className="w-full text-right bg-white rounded-lg border border-gray-200 p-3 text-sm hover:border-black">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-bold" dir="ltr">{o.orderNumber}</p>
                        <p className="text-xs text-gray-500">{o.customerName} • {o.phone}</p>
                      </div>
                      <div className="text-left">
                        <p className="font-bold text-xs">{(o.total ?? 0).toFixed(2)} ج.م</p>
                        <span className="text-[10px] bg-gray-100 px-2 py-0.5 rounded-full">{o.status}</span>
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}
          {results.customers.length === 0 && results.drivers.length === 0 && results.orders.length === 0 && (
            <p className="text-center text-gray-400 text-sm py-8">لا نتائج</p>
          )}
        </div>
      )}
    </div>
  )
}
