import { useState, useEffect } from 'react'
import { MapPin, Loader2, Plus, Power, X } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminZonesPage() {
  const [zones, setZones] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ name: '', nameAr: '', centerLat: 30.05, centerLng: 31.36, radiusM: 3000 })

  const load = () => {
    setLoading(true)
    adminAPI.getZones().then((r) => setZones(r.zones || [])).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await adminAPI.createZone(form)
      toast.success('تم إنشاء المنطقة')
      setShowCreate(false)
      setForm({ name: '', nameAr: '', centerLat: 30.05, centerLng: 31.36, radiusM: 3000 })
      load()
    } catch (err: any) { toast.error(err.message) }
  }

  const toggle = async (z: any) => {
    try { await adminAPI.updateZone(z.id, { IsActive: !z.isActive } as any); load() }
    catch (e: any) { toast.error(e.message) }
  }

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">مناطق العمل</h1>
        <button onClick={() => setShowCreate(true)} className="px-3 h-9 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" /> منطقة جديدة
        </button>
      </div>
      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       <div className="grid md:grid-cols-2 gap-3">
         {zones.map((z) => (
           <div key={z.id} className="bg-white rounded-lg border border-gray-200 p-4">
             <div className="flex items-start justify-between mb-2">
               <div>
                 <p className="font-bold flex items-center gap-2"><MapPin className="w-4 h-4" /> {z.nameAr}</p>
                 <p className="text-xs text-gray-400">{z.name}</p>
               </div>
               <span className={`text-[10px] px-2 py-0.5 rounded-full ${z.isActive ? 'bg-black text-white' : 'bg-gray-200 text-gray-500'}`}>
                 {z.isActive ? 'مفعّل' : 'موقوف'}
               </span>
             </div>
             <div className="text-xs text-gray-600 space-y-1 mb-3">
               <div>المركز: {(z.centerLat ?? 0).toFixed(4)}, {(z.centerLng ?? 0).toFixed(4)}</div>
               <div>نصف القطر: {z.radiusM} م</div>
             </div>
             <button onClick={() => toggle(z)} className="w-full h-8 rounded-lg border border-gray-200 hover:bg-gray-50 text-xs font-medium flex items-center justify-center gap-1.5">
               <Power className="w-3.5 h-3.5" /> {z.isActive ? 'إيقاف' : 'تفعيل'}
             </button>
           </div>
         ))}
       </div>}

      {showCreate && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4" onClick={(e) => e.target === e.currentTarget && setShowCreate(false)}>
          <div className="bg-white rounded-xl w-full max-w-md p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-bold">منطقة جديدة</h3>
              <button onClick={() => setShowCreate(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="space-y-3">
              <input required placeholder="الاسم بالعربية" value={form.nameAr} onChange={(e) => setForm({...form, nameAr: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="الاسم بالإنجليزية" value={form.name} onChange={(e) => setForm({...form, name: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <div className="grid grid-cols-2 gap-2">
                <input type="number" step="0.0001" placeholder="خط العرض" value={form.centerLat} onChange={(e) => setForm({...form, centerLat: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" step="0.0001" placeholder="خط الطول" value={form.centerLng} onChange={(e) => setForm({...form, centerLng: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>
              <input type="number" placeholder="نصف القطر (متر)" value={form.radiusM} onChange={(e) => setForm({...form, radiusM: +e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <button type="submit" className="w-full h-11 rounded-lg bg-black text-white font-medium hover:bg-gray-800">إنشاء</button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
