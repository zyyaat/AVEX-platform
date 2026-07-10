import { useState, useEffect } from 'react'
import { Settings, Loader2, Save } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

const configKeys = [
  { key: 'dispatch_radius_m', label: 'نطاق التوزيع (متر)', desc: 'أقصى مسافة مندوب↔مطعم للسماح بالتوزيع' },
  { key: 'offer_expiry_seconds', label: 'انتهاء العرض (ثانية)', desc: 'مدة عرض الطلب على المندوب قبل انتهائه' },
  { key: 'pickup_geofence_m', label: 'Geofence الاستلام (متر)', desc: 'المسافة المطلوبة من المطعم لتأكيد الاستلام' },
  { key: 'delivery_geofence_m', label: 'Geofence التسليم (متر)', desc: 'المسافة المطلوبة من العميل لتأكيد التسليم' },
  { key: 'location_stale_seconds', label: 'انتهاء موقع المندوب (ثانية)', desc: 'يعتبر المندوب غير متصل إذا لم يحدّث موقعه خلال هذه المدة' },
  { key: 'delivery_fee', label: 'رسوم التوصيل الافتراضية (ج.م)', desc: 'رسوم توصيل افتراضية للعميل' },
  { key: 'free_shipping_threshold', label: 'حد التوصيل المجاني (ج.م)', desc: 'توصيل مجاني للعميل فوق هذا المبلغ' },
]

export default function AdminSettingsPage() {
  const [values, setValues] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/v1/admin/settings', { headers: { Authorization: `Bearer ${localStorage.getItem('avex_admin_token')}` } }).then(r => r.json()).then(d => {
      setValues(d.settings || {})
      setLoading(false)
    })
  }, [])

  const save = async (key: string) => {
    try {
      await adminAPI.updateSetting(key, values[key])
      toast.success('تم الحفظ')
    } catch (e: any) { toast.error(e.message) }
  }

  if (loading) return <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">الإعدادات</h1>
      <div className="space-y-3 max-w-2xl">
        {configKeys.map((c) => (
          <div key={c.key} className="bg-white rounded-lg border border-gray-200 p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1">
                <label className="font-bold text-sm block">{c.label}</label>
                <p className="text-xs text-gray-500 mt-0.5">{c.desc}</p>
                <input value={values[c.key] || ''} onChange={(e) => setValues({...values, [c.key]: e.target.value})}
                  className="mt-2 w-full h-10 px-3 rounded-lg border border-gray-200 text-sm focus:outline-none focus:border-black" />
              </div>
              <button onClick={() => save(c.key)} className="px-3 h-10 rounded-lg bg-black text-white text-sm font-medium flex items-center gap-1.5 flex-shrink-0">
                <Save className="w-3.5 h-3.5" /> حفظ
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
