import { useState, useEffect } from 'react'
import { Store, Loader2, Plus, Power, MapPin, X, Phone } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminRestaurantsPage() {
  const [rests, setRests] = useState<any[]>([])
  const [zones, setZones] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ name: '', nameAr: '', descriptionAr: '', cuisines: '', zoneId: '', lat: 30.05, lng: 31.36, deliveryFee: 3.99, minOrder: 0, dtMin: 20, dtMax: 45, isPro: false })

  const load = () => {
    setLoading(true)
    Promise.all([adminAPI.getRestaurants(), adminAPI.getZones()])
      .then(([r, z]) => { setRests(r.restaurants || []); setZones(z.zones || []) })
      .finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const r = await adminAPI.createRestaurant(form)
      toast.success(`تم إنشاء المطعم. حساب التاجر: ${r.merchantPhone} / ${r.merchantPassword}`)
      setShowCreate(false)
      setForm({ name: '', nameAr: '', descriptionAr: '', cuisines: '', zoneId: '', lat: 30.05, lng: 31.36, deliveryFee: 3.99, minOrder: 0, dtMin: 20, dtMax: 45, isPro: false })
      load()
    } catch (err: any) { toast.error(err.message) }
  }

  const toggleActive = async (r: any) => {
    try { await adminAPI.updateRestaurant(r.id, { isActive: !r.isActive }); load() }
    catch (e: any) { toast.error(e.message) }
  }

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">المطاعم</h1>
        <button onClick={() => setShowCreate(true)} className="px-3 h-9 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" /> مطعم جديد
        </button>
      </div>

      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       <div className="grid md:grid-cols-2 gap-3">
         {rests.map((r) => (
           <div key={r.id} className="bg-white rounded-lg border border-gray-200 p-4">
             <div className="flex items-start justify-between mb-2">
               <div>
                 <p className="font-bold">{r.nameAr}</p>
                 <p className="text-xs text-gray-400">{r.name}</p>
               </div>
               <span className={`text-[10px] px-2 py-0.5 rounded-full ${r.isActive ? 'bg-black text-white' : 'bg-gray-200 text-gray-500'}`}>
                 {r.isActive ? 'مفعّل' : 'موقوف'}
               </span>
             </div>
             <div className="space-y-1 text-xs text-gray-600 mb-3">
               <div className="flex items-center gap-1"><MapPin className="w-3 h-3" /> {r.zoneName || 'بدون منطقة'}</div>
               <div>الأصناف: {r.menuCount} • طلبات اليوم: {r.todayOrders}</div>
               <div>رسوم التوصيل: {r.deliveryFee} ج.م • تقييم: {(r.rating ?? 0).toFixed(1)} ({r.ratingCount})</div>
             </div>
             <button onClick={() => toggleActive(r)}
               className="w-full h-8 rounded-lg border border-gray-200 hover:bg-gray-50 text-xs font-medium flex items-center justify-center gap-1.5">
               <Power className="w-3.5 h-3.5" /> {r.isActive ? 'إيقاف' : 'تفعيل'}
             </button>
           </div>
         ))}
       </div>}

      {showCreate && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4" onClick={(e) => e.target === e.currentTarget && setShowCreate(false)}>
          <div className="bg-white rounded-xl w-full max-w-md max-h-[90vh] overflow-y-auto p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-bold">مطعم جديد</h3>
              <button onClick={() => setShowCreate(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="space-y-3">
              <input required placeholder="الاسم بالعربية" value={form.nameAr} onChange={(e) => setForm({...form, nameAr: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="الاسم بالإنجليزية" value={form.name} onChange={(e) => setForm({...form, name: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="الوصف" value={form.descriptionAr} onChange={(e) => setForm({...form, descriptionAr: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="أنواع المطبخ (مثال: برغر, بيتزا)" value={form.cuisines} onChange={(e) => setForm({...form, cuisines: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <select required value={form.zoneId} onChange={(e) => setForm({...form, zoneId: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 bg-white focus:outline-none focus:border-black">
                <option value="">اختر المنطقة</option>
                {zones.map((z) => <option key={z.id} value={z.id}>{z.nameAr}</option>)}
              </select>
              <div className="grid grid-cols-2 gap-2">
                <input type="number" step="0.0001" placeholder="lat" value={form.lat} onChange={(e) => setForm({...form, lat: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" step="0.0001" placeholder="lng" value={form.lng} onChange={(e) => setForm({...form, lng: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <input type="number" step="0.01" placeholder="رسوم التوصيل" value={form.deliveryFee} onChange={(e) => setForm({...form, deliveryFee: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" step="0.01" placeholder="الحد الأدنى للطلب" value={form.minOrder} onChange={(e) => setForm({...form, minOrder: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <input type="number" placeholder="زمن التوصيل (دقيقة)" value={form.dtMin} onChange={(e) => setForm({...form, dtMin: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" placeholder="أقصى زمن (دقيقة)" value={form.dtMax} onChange={(e) => setForm({...form, dtMax: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>
              <button type="submit" className="w-full h-11 rounded-lg bg-black text-white font-medium hover:bg-gray-800">إنشاء</button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
