import { useState, useEffect } from 'react'
import { Bike, Loader2, Power, Award, Phone, MapPin, Plus, X, CheckCircle2 } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminDriversPage() {
  const [drivers, setDrivers] = useState<any[]>([])
  const [tiers, setTiers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({
    name: '', phone: '', password: '', vehicle_type: 'motorcycle',
    license_number: '', national_id: '', license_plate: '', zone_ids: 'zone-cairo',
  })

  const load = () => {
    setLoading(true)
    Promise.all([adminAPI.getDrivers(), adminAPI.getTiers().catch(() => ({ tiers: [] }))])
      .then(([d, t]) => {
        setDrivers(d || [])
        setTiers((t as any).tiers || [])
      })
      .finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const toggleActive = async (d: any) => {
    try { await adminAPI.updateDriverStatus(d.id, !d.isActive); load() }
    catch (e: any) { toast.error(e.message) }
  }
  const changeTier = async (d: any, tierId: string) => {
    try { await adminAPI.updateDriverTier(d.id, tierId); load(); toast.success('تم تحديث المستوى') }
    catch (e: any) { toast.error(e.message) }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    try {
      // Single call — backend creates both identity + dispatch in one request
      const result = await adminAPI.createDriver({
        name: form.name,
        phone: form.phone,
        password: form.password,
        vehicle_type: form.vehicle_type,
        license_number: form.license_number || `LIC-${Date.now()}`,
        national_id: form.national_id || `ID-${Date.now()}`,
        license_plate: form.license_plate,
        zone_ids: form.zone_ids.split(',').map(z => z.trim()).filter(Boolean),
      })

      toast.success(`تم إنشاء المندوب بنجاح!`)

      setShowCreate(false)
      setForm({ name: '', phone: '', password: '', vehicle_type: 'motorcycle', license_number: '', national_id: '', license_plate: '', zone_ids: 'zone-cairo' })
      load()
    } catch (err: any) {
      toast.error(err.message || 'فشل إنشاء المندوب')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">المندوبين ({drivers.length})</h1>
        <button onClick={() => setShowCreate(true)}
          className="px-3 h-9 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" /> إضافة مندوب
        </button>
      </div>

      {loading ? (
        <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
      ) : drivers.length === 0 ? (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
          <Bike className="w-12 h-12 text-gray-300 mx-auto mb-2" />
          <p className="text-sm text-gray-400">لا يوجد مندوبين مسجلين</p>
          <p className="text-xs text-gray-400 mt-1">اضغط "إضافة مندوب" لإنشاء أول مندوب</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-600 text-xs">
              <tr>
                <th className="px-3 py-2 text-right">المندوب</th>
                <th className="px-3 py-2 text-right">الهاتف</th>
                <th className="px-3 py-2 text-right">المركبة</th>
                <th className="px-3 py-2 text-right">الحالة</th>
                <th className="px-3 py-2 text-right">متصل</th>
                <th className="px-3 py-2 text-right">طلبات</th>
                <th className="px-3 py-2 text-right">التقييم</th>
                <th className="px-3 py-2 text-right">إجراءات</th>
              </tr>
            </thead>
            <tbody>
              {drivers.map((d) => (
                <tr key={d.id} className="border-t border-gray-100">
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 rounded-full bg-orange-100 flex items-center justify-center text-orange-600 text-xs font-bold">
                        {(d.userId || d.id || '?').charAt(0).toUpperCase()}
                      </div>
                      <span className="font-medium text-xs">{d.userId ? `ID: ${d.userId.slice(0, 8)}...` : d.id}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-xs text-gray-500" dir="ltr">—</td>
                  <td className="px-3 py-2 text-xs">{d.vehicleType || d.vehicle_type || '—'}</td>
                  <td className="px-3 py-2">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full ${d.status === 'suspended' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                      {d.status === 'suspended' ? 'موقوف' : 'مفعّل'}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    {d.status === 'online' || d.status === 'busy' ?
                      <span className="text-[10px] text-green-600 font-bold">● متصل</span> :
                      <span className="text-[10px] text-gray-400">○ غير متصل</span>}
                  </td>
                  <td className="px-3 py-2 text-xs">{d.totalDeliveries ?? d.total_deliveries ?? 0}</td>
                  <td className="px-3 py-2 text-xs">{(d.rating ?? 5).toFixed(1)} ⭐</td>
                  <td className="px-3 py-2">
                    <button onClick={() => toggleActive(d)} className="text-xs px-2 py-1 rounded border border-gray-200 hover:bg-gray-50">
                      <Power className="w-3 h-3 inline" /> {d.status === 'suspended' ? 'تفعيل' : 'إيقاف'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Driver Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-2xl w-full max-w-md p-6 shadow-2xl max-h-[90dvh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-bold text-lg flex items-center gap-2">
                <Bike className="w-5 h-5 text-orange-500" /> إضافة مندوب جديد
              </h2>
              <button onClick={() => setShowCreate(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreate} className="space-y-3">
              <div>
                <label className="text-xs text-gray-500">الاسم *</label>
                <input type="text" value={form.name} onChange={(e) => setForm({...form, name: e.target.value})} required
                  className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
              </div>
              <div>
                <label className="text-xs text-gray-500">رقم الهاتف *</label>
                <input type="tel" dir="ltr" value={form.phone} onChange={(e) => setForm({...form, phone: e.target.value})} required
                  placeholder="01xxxxxxxxx" className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm text-right" />
              </div>
              <div>
                <label className="text-xs text-gray-500">كلمة المرور *</label>
                <input type="text" value={form.password} onChange={(e) => setForm({...form, password: e.target.value})} required
                  placeholder="12345678" className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
              </div>
              <div>
                <label className="text-xs text-gray-500">نوع المركبة</label>
                <select value={form.vehicle_type} onChange={(e) => setForm({...form, vehicle_type: e.target.value})}
                  className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm">
                  <option value="motorcycle">دراجة بخارية</option>
                  <option value="car">سيارة</option>
                  <option value="scooter">سكوتر</option>
                  <option value="bike">دراجة</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="text-xs text-gray-500">رقم الرخصة</label>
                  <input type="text" value={form.license_number} onChange={(e) => setForm({...form, license_number: e.target.value})}
                    className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
                </div>
                <div>
                  <label className="text-xs text-gray-500">رقم البطاقة</label>
                  <input type="text" value={form.national_id} onChange={(e) => setForm({...form, national_id: e.target.value})}
                    className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
                </div>
              </div>
              <div>
                <label className="text-xs text-gray-500">رقم اللوحة</label>
                <input type="text" value={form.license_plate} onChange={(e) => setForm({...form, license_plate: e.target.value})}
                  placeholder="ABC-123" className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
              </div>
              <div>
                <label className="text-xs text-gray-500">المناطق (مفصولة بفواصل)</label>
                <input type="text" value={form.zone_ids} onChange={(e) => setForm({...form, zone_ids: e.target.value})}
                  placeholder="zone-cairo" className="w-full h-10 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black text-sm" />
              </div>

              <button type="submit" disabled={creating}
                className="w-full h-11 rounded-lg bg-black text-white font-medium flex items-center justify-center gap-2 disabled:opacity-50">
                {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
                إنشاء المندوب
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
