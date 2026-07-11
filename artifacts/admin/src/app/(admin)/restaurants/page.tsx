import { useState, useEffect, useRef } from 'react'
import { Store, Loader2, Plus, Power, MapPin, X, Phone, Navigation, CheckCircle2 } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminRestaurantsPage() {
  const [rests, setRests] = useState<any[]>([])
  const [zones, setZones] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [mapPickMode, setMapPickMode] = useState(false)
  const [form, setForm] = useState({
    name: '', nameAr: '', descriptionAr: '', cuisines: '', zoneId: 'zone-cairo',
    lat: 30.0444, lng: 31.2357, deliveryFee: 3.99, minOrder: 0, dtMin: 20, dtMax: 45, isPro: false,
    merchantPhone: '', merchantPassword: '',
  })
  const mapRef = useRef<any>(null)
  const mapContainerRef = useRef<HTMLDivElement>(null)
  const markerRef = useRef<any>(null)

  const load = () => {
    setLoading(true)
    Promise.all([
      adminAPI.getRestaurants(),
      adminAPI.getZones().catch(() => []),
    ]).then(([r, z]) => {
      setRests(r || [])
      setZones(z || [])
    }).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  // ===== Map for location picker =====
  useEffect(() => {
    if (!showCreate || !mapContainerRef.current) return
    if (mapRef.current) return

    let cancelled = false

    if (!document.querySelector('#leaflet-css')) {
      const link = document.createElement('link')
      link.id = 'leaflet-css'
      link.rel = 'stylesheet'
      link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css'
      document.head.appendChild(link)
    }

    const initMap = (L: any) => {
      if (cancelled || !mapContainerRef.current) return
      const map = L.map(mapContainerRef.current, {
        center: [form.lat, form.lng], zoom: 13, zoomControl: false, attributionControl: false,
      })
      L.tileLayer('https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png', {
        maxZoom: 19, subdomains: 'abcd',
      }).addTo(map)

      const icon = L.divIcon({
        className: 'rest-marker',
        html: `<div style="width:28px;height:28px;border-radius:50%;background:#FF6B35;border:3px solid white;box-shadow:0 2px 6px rgba(0,0,0,0.3);display:flex;align-items:center;justify-content:center;">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="white"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>
        </div>`,
        iconSize: [28, 28], iconAnchor: [14, 14],
      })
      markerRef.current = L.marker([form.lat, form.lng], { icon }).addTo(map)

      map.on('click', (e: any) => {
        const { lat, lng } = e.latlng
        markerRef.current.setLatLng([lat, lng])
        setForm(f => ({ ...f, lat: lat.toFixed(6), lng: lng.toFixed(6) }))
      })

      mapRef.current = map
      setTimeout(() => map.invalidateSize(), 100)
    }

    if ((window as any).L) { initMap((window as any).L) }
    else {
      const script = document.createElement('script')
      script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js'
      script.onload = () => { if (!cancelled && (window as any).L) initMap((window as any).L) }
      document.head.appendChild(script)
    }

    return () => {
      cancelled = true
      if (mapRef.current) { mapRef.current.remove(); mapRef.current = null }
    }
  }, [showCreate])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    try {
      const result = await adminAPI.createRestaurant({
        name: form.name || form.nameAr,
        name_ar: form.nameAr,
        description_ar: form.descriptionAr,
        cuisines: form.cuisines,
        zone_id: form.zoneId,
        lat: Number(form.lat), lng: Number(form.lng),
        delivery_fee: Number(form.deliveryFee),
        min_order: Number(form.minOrder),
        delivery_time_min: Number(form.dtMin),
        delivery_time_max: Number(form.dtMax),
        is_pro: form.isPro,
      })
      toast.success(`تم إنشاء المطعم بنجاح!`)
      setShowCreate(false)
      setForm({ name: '', nameAr: '', descriptionAr: '', cuisines: '', zoneId: 'zone-cairo', lat: 30.0444, lng: 31.2357, deliveryFee: 3.99, minOrder: 0, dtMin: 20, dtMax: 45, isPro: false, merchantPhone: '', merchantPassword: '' })
      if (mapRef.current) { mapRef.current.remove(); mapRef.current = null }
      load()
    } catch (err: any) {
      toast.error(err.message || 'فشل إنشاء المطعم')
    } finally {
      setCreating(false)
    }
  }

  const toggleActive = async (r: any) => {
    try {
      await adminAPI.updateRestaurant(r.id, { is_active: !r.isActive })
      load()
    } catch (e: any) { toast.error(e.message) }
  }

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">المطاعم ({rests.length})</h1>
        <button onClick={() => setShowCreate(true)}
          className="px-3 h-9 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" /> مطعم جديد
        </button>
      </div>

      {loading ? (
        <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
      ) : rests.length === 0 ? (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
          <Store className="w-12 h-12 text-gray-300 mx-auto mb-2" />
          <p className="text-sm text-gray-400">لا يوجد مطاعم مسجلة</p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-3">
          {rests.map((r: any) => (
            <div key={r.id} className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-start justify-between mb-2">
                <div>
                  <p className="font-bold">{r.nameAr || r.nameAr || r.name}</p>
                  <p className="text-xs text-gray-400">{r.name}</p>
                </div>
                <span className={`text-[10px] px-2 py-0.5 rounded-full ${r.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-500'}`}>
                  {r.isActive ? 'مفعّل' : 'موقوف'}
                </span>
              </div>
              <div className="space-y-1 text-xs text-gray-600 mb-3">
                <div className="flex items-center gap-1"><MapPin className="w-3 h-3" /> {r.zoneId || 'بدون منطقة'}</div>
                <div>رسوم التوصيل: {(r.deliveryFee ?? 0).toFixed(2)} ج.م</div>
                <div>تقييم: {(r.rating ?? 0).toFixed(1)} ⭐</div>
                <div className="font-mono text-gray-400" dir="ltr">{r.lat?.toFixed(4)}, {r.lng?.toFixed(4)}</div>
              </div>
              <button onClick={() => toggleActive(r)}
                className="w-full h-8 rounded-lg border border-gray-200 hover:bg-gray-50 text-xs font-medium flex items-center justify-center gap-1.5">
                <Power className="w-3.5 h-3.5" /> {r.isActive ? 'إيقاف' : 'تفعيل'}
              </button>
            </div>
          ))}
        </div>
      )}

      {/* ===== Create Restaurant Modal ===== */}
      {showCreate && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
          onClick={(e) => e.target === e.currentTarget && setShowCreate(false)}>
          <div className="bg-white rounded-xl w-full max-w-md max-h-[90vh] overflow-y-auto p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-bold">مطعم جديد</h3>
              <button onClick={() => setShowCreate(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreate} className="space-y-3">
              {/* Map for location */}
              <div>
                <label className="text-xs text-gray-500 mb-1 block">موقع المطعم — اضغط على الخريطة</label>
                <div ref={mapContainerRef} className="w-full h-48 rounded-lg border border-gray-200 overflow-hidden" />
                <div className="flex gap-2 mt-1">
                  <input type="number" step="0.000001" value={form.lat} onChange={(e) => setForm({...form, lat: +e.target.value})}
                    className="flex-1 h-9 px-2 rounded border border-gray-200 text-xs" placeholder="lat" />
                  <input type="number" step="0.000001" value={form.lng} onChange={(e) => setForm({...form, lng: +e.target.value})}
                    className="flex-1 h-9 px-2 rounded border border-gray-200 text-xs" placeholder="lng" />
                </div>
              </div>

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
                {zones.map((z: any) => (
                  <option key={z.id} value={z.id}>{z.nameAr || z.name}</option>
                ))}
              </select>
              <div className="grid grid-cols-2 gap-2">
                <input type="number" step="0.01" placeholder="رسوم التوصيل" value={form.deliveryFee} onChange={(e) => setForm({...form, deliveryFee: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" step="0.01" placeholder="الحد الأدنى" value={form.minOrder} onChange={(e) => setForm({...form, minOrder: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <input type="number" placeholder="زمن (دقيقة)" value={form.dtMin} onChange={(e) => setForm({...form, dtMin: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
                <input type="number" placeholder="أقصى زمن" value={form.dtMax} onChange={(e) => setForm({...form, dtMax: +e.target.value})}
                  className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              </div>

              <button type="submit" disabled={creating}
                className="w-full h-11 rounded-lg bg-black text-white font-medium flex items-center justify-center gap-2 disabled:opacity-50">
                {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
                إنشاء المطعم
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
