import { useState, useEffect } from 'react'
import { Bike, Loader2, Power, Award, Phone, MapPin } from 'lucide-react'
import { adminAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function AdminDriversPage() {
  const [drivers, setDrivers] = useState<any[]>([])
  const [tiers, setTiers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    Promise.all([adminAPI.getDrivers(), adminAPI.getTiers()])
      .then(([d, t]) => { setDrivers(d.drivers || []); setTiers(t.tiers || []) })
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

  return (
    <div dir="rtl">
      <h1 className="text-xl font-bold mb-4">المندوبين ({drivers.length})</h1>
      {loading ? <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div> :
       <div className="bg-white rounded-lg border border-gray-200 overflow-x-auto">
         <table className="w-full text-sm">
           <thead className="bg-gray-50 text-gray-600 text-xs">
             <tr>
               <th className="px-3 py-2 text-right">المندوب</th>
               <th className="px-3 py-2 text-right">الهاتف</th>
               <th className="px-3 py-2 text-right">المستوى</th>
               <th className="px-3 py-2 text-right">الحالة</th>
               <th className="px-3 py-2 text-right">متصل</th>
               <th className="px-3 py-2 text-right">طلبات</th>
               <th className="px-3 py-2 text-right">أرباح</th>
               <th className="px-3 py-2 text-right">إجراءات</th>
             </tr>
           </thead>
           <tbody>
             {drivers.map((d) => (
               <tr key={d.id} className="border-t border-gray-100">
                 <td className="px-3 py-2">
                   <div className="flex items-center gap-2">
                     <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold`} style={{ backgroundColor: d.tierColor || '#000' }}>
                       {d.name?.charAt(0)}
                     </div>
                     <span className="font-medium text-xs">{d.name}</span>
                   </div>
                 </td>
                 <td className="px-3 py-2 text-xs" dir="ltr">{d.phone}</td>
                 <td className="px-3 py-2">
                   <select value={d.tierId || ''} onChange={(e) => changeTier(d, e.target.value)}
                     className="text-xs px-2 py-1 rounded border border-gray-200 bg-white">
                     {tiers.map((t) => <option key={t.id} value={t.id}>{t.nameAr}</option>)}
                   </select>
                 </td>
                 <td className="px-3 py-2">
                   <span className={`text-[10px] px-2 py-0.5 rounded-full ${d.isActive ? 'bg-black text-white' : 'bg-gray-200 text-gray-500'}`}>
                     {d.isActive ? 'مفعّل' : 'موقوف'}
                   </span>
                   {!d.isVerified && <span className="text-[10px] px-2 py-0.5 rounded-full bg-gray-100 text-gray-500 mr-1">غير موثّق</span>}
                 </td>
                 <td className="px-3 py-2">
                   {d.isOnline ? <span className="text-[10px] text-black font-bold">● متصل</span> : <span className="text-[10px] text-gray-400">○ غير متصل</span>}
                 </td>
                 <td className="px-3 py-2 text-xs">{d.completedOrders}</td>
                 <td className="px-3 py-2 text-xs font-bold">{(d.totalEarnings ?? 0).toFixed(2)}</td>
                 <td className="px-3 py-2">
                   <button onClick={() => toggleActive(d)} className="text-xs px-2 py-1 rounded border border-gray-200 hover:bg-gray-50">
                     <Power className="w-3 h-3 inline" /> {d.isActive ? 'إيقاف' : 'تفعيل'}
                   </button>
                 </td>
               </tr>
             ))}
           </tbody>
         </table>
       </div>}
    </div>
  )
}
