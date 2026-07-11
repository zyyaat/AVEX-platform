import { useState, useEffect, useRef } from 'react'
import { MapPin, Loader2, Plus, X, Trash2, CheckCircle2 } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminZonesPage() {
  const [zones, setZones] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({
    id: '', name: '', nameAr: '',
    centerLat: 30.0444, centerLng: 31.2357, radiusM: 3000,
  })
  const mapRef = useRef<any>(null)
  const mapContainerRef = useRef<HTMLDivElement>(null)
  const circleRef = useRef<any>(null)
  const markerRef = useRef<any>(null)

  const load = () => {
    setLoading(true)
    adminAPI.getZones().then((r) => setZones(r || [])).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  // ===== Map for zone drawing =====
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
        center: [form.centerLat, form.centerLng], zoom: 12, zoomControl: false, attributionControl: false,
      })
      L.tileLayer('https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png', {
        maxZoom: 19, subdomains: 'abcd',
      }).addTo(map)

      // Center marker
      const icon = L.divIcon({
        className: 'zone-marker',
        html: `<div style="width:20px;height:20px;border-radius:50%;background:#FF6B35;border:3px solid white;box-shadow:0 2px 6px rgba(0,0,0,0.3);"></div>`,
        iconSize: [20, 20], iconAnchor: [10, 10],
      })
      markerRef.current = L.marker([form.centerLat, form.centerLng], { icon, draggable: true }).addTo(map)

      // Circle showing zone radius
      circleRef.current = L.circle([form.centerLat, form.centerLng], {
        radius: form.radiusM,
        color: '#FF6B35', fillColor: '#FF6B35', fillOpacity: 0.15, weight: 2,
      }).addTo(map)

      // Click to move center
      map.on('click', (e: any) => {
        const { lat, lng } = e.latlng
        markerRef.current.setLatLng([lat, lng])
        circleRef.current.setLatLng([lat, lng])
        setForm(f => ({ ...f, centerLat: lat.toFixed(6), centerLng: lng.toFixed(6) }))
      })

      // Drag marker to move center
      markerRef.current.on('drag', (e: any) => {
        const { lat, lng } = e.target.getLatLng()
        circleRef.current.setLatLng([lat, lng])
        setForm(f => ({ ...f, centerLat: lat.toFixed(6), centerLng: lng.toFixed(6) }))
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

  // Update circle when radius changes
  useEffect(() => {
    if (circleRef.current) {
      circleRef.current.setRadius(Number(form.radiusM))
    }
  }, [form.radiusM])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    try {
      await adminAPI.createZone({
        id: form.id || undefined,
        name: form.name,
        name_ar: form.nameAr,
        center_lat: Number(form.centerLat),
        center_lng: Number(form.centerLng),
        radius_m: Number(form.radiusM),
      })
      toast.success('تم إنشاء المنطقة')
      setShowCreate(false)
      setForm({ id: '', name: '', nameAr: '', centerLat: 30.0444, centerLng: 31.2357, radiusM: 3000 })
      if (mapRef.current) { mapRef.current.remove(); mapRef.current = null }
      load()
    } catch (err: any) { toast.error(err.message || 'فشل إنشاء المنطقة') }
    finally { setCreating(false) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('هل تريد حذف هذه المنطقة؟')) return
    try { await adminAPI.deleteZone(id); toast.success('تم الحذف'); load() }
    catch (e: any) { toast.error(e.message) }
  }

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">المناطق ({zones.length})</h1>
        <button onClick={() => setShowCreate(true)}
          className="px-3 h-9 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" /> منطقة جديدة
        </button>
      </div>

      {loading ? (
        <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
      ) : zones.length === 0 ? (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
          <MapPin className="w-12 h-12 text-gray-300 mx-auto mb-2" />
          <p className="text-sm text-gray-400">لا يوجد مناطق مسجلة</p>
          <p className="text-xs text-gray-400 mt-1">اضغط "منطقة جديدة" لرسم أول منطقة</p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-3">
          {zones.map((z: any) => (
            <div key={z.id} className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-start justify-between mb-2">
                <div>
                  <p className="font-bold">{z.nameAr || z.name}</p>
                  <p className="text-xs text-gray-400">{z.id}</p>
                </div>
                <button onClick={() => handleDelete(z.id)}
                  className="w-7 h-7 rounded-full hover:bg-red-50 text-red-500 flex items-center justify-center">
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
              <div className="space-y-1 text-xs text-gray-600">
                <div className="flex items-center gap-1"><MapPin className="w-3 h-3" />
                  المركز: {Number(z.centerLat || z.center_lat).toFixed(4)}, {Number(z.centerLng || z.center_lng).toFixed(4)}
                </div>
                <div>نصف القطر: {(z.radiusM || z.radius_m || 0).toLocaleString()} متر</div>
                <span className={`inline-block text-[10px] px-2 py-0.5 rounded-full ${z.isActive || z.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-500'}`}>
                  {z.isActive || z.is_active ? 'مفعّل' : 'موقوف'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ===== Create Zone Modal ===== */}
      {showCreate && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
          onClick={(e) => e.target === e.currentTarget && setShowCreate(false)}>
          <div className="bg-white rounded-xl w-full max-w-md max-h-[90vh] overflow-y-auto p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-bold">منطقة جديدة — ارسم على الخريطة</h3>
              <button onClick={() => setShowCreate(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreate} className="space-y-3">
              {/* Map */}
              <div>
                <label className="text-xs text-gray-500 mb-1 block">اضغط على الخريطة لتحديد المركز — اسحب الدائرة لتكبيرها</label>
                <div ref={mapContainerRef} className="w-full h-56 rounded-lg border border-gray-200 overflow-hidden" />
              </div>

              {/* Radius slider */}
              <div>
                <label className="text-xs text-gray-500 mb-1 block">نصف القطر: {Number(form.radiusM).toLocaleString()} متر</label>
                <input type="range" min="500" max="10000" step="500" value={form.radiusM}
                  onChange={(e) => setForm({...form, radiusM: +e.target.value})}
                  className="w-full h-2 rounded-lg appearance-none bg-gray-200 cursor-pointer" />
              </div>

              {/* Center coords */}
              <div className="grid grid-cols-2 gap-2">
                <input type="number" step="0.000001" value={form.centerLat} onChange={(e) => setForm({...form, centerLat: +e.target.value})}
                  className="w-full h-9 px-2 rounded border border-gray-200 text-xs" placeholder="lat" />
                <input type="number" step="0.000001" value={form.centerLng} onChange={(e) => setForm({...form, centerLng: +e.target.value})}
                  className="w-full h-9 px-2 rounded border border-gray-200 text-xs" placeholder="lng" />
              </div>

              <input required placeholder="اسم المنطقة (إنجليزي)" value={form.name} onChange={(e) => setForm({...form, name: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="الاسم بالعربية" value={form.nameAr} onChange={(e) => setForm({...form, nameAr: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />
              <input placeholder="المعرف (اختياري — zone-cairo)" value={form.id} onChange={(e) => setForm({...form, id: e.target.value})}
                className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black" />

              <button type="submit" disabled={creating}
                className="w-full h-11 rounded-lg bg-black text-white font-medium flex items-center justify-center gap-2 disabled:opacity-50">
                {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
                إنشاء المنطقة
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
