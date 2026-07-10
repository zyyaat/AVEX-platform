
import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { ArrowRight, MapPin, CreditCard, Plus, Trash2, Home, Check, Loader2, User, Lock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { useAuth } from '@/store/auth'
import { userAPI } from '@/lib/api'
import { toast } from 'sonner'

interface AccountPageProps { onBack: () => void; onLoginRequired: () => void }

export function AccountPage({ onBack, onLoginRequired }: AccountPageProps) {
  const { isAuthenticated, user, logout, initialize } = useAuth()
  const [addresses, setAddresses] = useState<any[]>([])
  const [cards, setCards] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddAddress, setShowAddAddress] = useState(false)
  const [showAddCard, setShowAddCard] = useState(false)
  const [authChecked, setAuthChecked] = useState(false)

  useEffect(() => { initialize().then(() => setAuthChecked(true)) }, [initialize])
  useEffect(() => {
    if (!authChecked) return
    if (!isAuthenticated) { onLoginRequired(); return }
    loadData()
  }, [isAuthenticated, authChecked])

  const loadData = async () => {
    try {
      const [addrRes, cardRes] = await Promise.all([userAPI.getAddresses(), userAPI.getCards()])
      setAddresses(addrRes.addresses || [])
      setCards(cardRes.cards || [])
    } catch {} finally { setLoading(false) }
  }

  if (!isAuthenticated) return <div className="min-h-dvh bg-gray-50 flex items-center justify-center p-4"><div className="text-center bg-white rounded-lg border p-8"><Button onClick={onBack} className="rounded-lg bg-black">العودة</Button></div></div>

  return (
    <div className="min-h-dvh bg-gray-50" dir="rtl">
      <header className="sticky top-0 z-30 bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 h-14 flex items-center justify-between">
          <button onClick={onBack} className="flex items-center gap-2 text-sm font-medium text-gray-700"><ArrowRight className="w-4 h-4" /> العودة</button>
          <h1 className="font-bold text-base text-gray-900">حسابي</h1>
          <Button onClick={() => { logout(); onBack() }} variant="outline" size="sm" className="rounded-lg">خروج</Button>
        </div>
      </header>
      <div className="container mx-auto px-4 py-6 max-w-2xl space-y-5">
        <div className="bg-white rounded-lg border p-5 flex items-center gap-4">
          <div className="w-14 h-14 rounded-full bg-gradient-to-br from-black to-gray-800 flex items-center justify-center"><span className="text-2xl font-bold text-white">{user?.name?.charAt(0) ?? "?"}</span></div>
          <div className="flex-1"><p className="font-bold text-lg">{user?.name}</p><p className="text-sm text-gray-500" dir="ltr">{user?.phone}</p></div>
          {user?.loyaltyPoints ? <div className="text-center"><p className="text-2xl font-bold text-black">{user.loyaltyPoints}</p><p className="text-[10px] text-gray-500">نقطة ولاء</p></div> : null}
        </div>
        {loading ? <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-black" /></div> : (
          <>
            <div className="bg-white rounded-lg border p-5">
              <div className="flex items-center justify-between mb-4"><h3 className="font-bold flex items-center gap-2"><MapPin className="w-5 h-5 text-black" /> عناويني</h3><Button onClick={() => setShowAddAddress(true)} size="sm" variant="outline" className="rounded-lg"><Plus className="w-4 h-4 ml-1" /> إضافة</Button></div>
              {addresses.length === 0 ? <p className="text-sm text-gray-500 text-center py-8">لا توجد عناوين محفوظة</p> : <div className="space-y-2">{addresses.map((addr) => <div key={addr.id} className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg"><div className="w-10 h-10 rounded-lg bg-gray-50 flex items-center justify-center"><Home className="w-5 h-5 text-black" /></div><div className="flex-1 min-w-0"><p className="font-medium text-sm">{addr.label}</p>{addr.addressText && <p className="text-xs text-gray-500 truncate">{addr.addressText}</p>}</div>{addr.isDefault && <span className="text-[10px] bg-gray-100 text-black px-2 py-0.5 rounded-full">افتراضي</span>}<button onClick={async () => { await userAPI.deleteAddress(addr.id); toast.success('تم حذف العنوان'); loadData() }} className="w-8 h-8 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-600 flex items-center justify-center"><Trash2 className="w-4 h-4" /></button></div>)}</div>}
            </div>
            <div className="bg-white rounded-lg border p-5">
              <div className="flex items-center justify-between mb-4"><h3 className="font-bold flex items-center gap-2"><CreditCard className="w-5 h-5 text-black" /> بطاقاتي</h3><Button onClick={() => setShowAddCard(true)} size="sm" variant="outline" className="rounded-lg"><Plus className="w-4 h-4 ml-1" /> إضافة</Button></div>
              {cards.length === 0 ? <div className="text-center py-8"><Lock className="w-10 h-10 text-gray-300 mx-auto mb-2" /><p className="text-sm text-gray-500">لا توجد بطاقات محفوظة</p></div> : <div className="space-y-2">{cards.map((card) => <div key={card.id} className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg"><div className="w-10 h-10 rounded-lg bg-gray-50 flex items-center justify-center"><CreditCard className="w-5 h-5 text-black" /></div><div className="flex-1"><p className="font-medium text-sm">{card.brand} •••• {card.last4}</p><p className="text-xs text-gray-500">{card.expMonth}/{card.expYear}</p></div>{card.isDefault ? <span className="text-[10px] bg-gray-100 text-black px-2 py-0.5 rounded-full flex items-center gap-0.5"><Check className="w-2.5 h-2.5" /> افتراضية</span> : <button onClick={async () => { await userAPI.setDefaultCard(card.id); toast.success('تم التعيين'); loadData() }} className="text-[10px] text-gray-500 hover:text-black">تعيين</button>}<button onClick={async () => { await userAPI.deleteCard(card.id); toast.success('تم الحذف'); loadData() }} className="w-8 h-8 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-600 flex items-center justify-center"><Trash2 className="w-4 h-4" /></button></div>)}</div>}
            </div>
          </>
        )}
      </div>
      {/* Add Address Dialog */}
      <AddAddressDialog open={showAddAddress} onOpenChange={setShowAddAddress} onSaved={() => { setShowAddAddress(false); loadData() }} />
      {/* Add Card Dialog */}
      <AddCardDialog open={showAddCard} onOpenChange={setShowAddCard} onSaved={() => { setShowAddCard(false); loadData() }} />
    </div>
  )
}

function AddAddressDialog({ open, onOpenChange, onSaved }: { open: boolean; onOpenChange: (v: boolean) => void; onSaved: () => void }) {
  const [label, setLabel] = useState('')
  const [addressText, setAddressText] = useState('')
  const [locating, setLocating] = useState(false)
  const [location, setLocation] = useState<{ lat: number; lng: number } | null>(null)
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    if (!label) { toast.error('أدخل اسم العنوان'); return }
    if (!location) { toast.error('حدد الموقع'); return }
    setSaving(true)
    try {
      await userAPI.saveAddress({ label, lat: location.lat, lng: location.lng, locationUrl: `https://www.google.com/maps?q=${location.lat},${location.lng}`, addressText, isDefault: false })
      toast.success('تم حفظ العنوان'); setLabel(''); setAddressText(''); setLocation(null); onSaved()
    } catch { toast.error('فشل الحفظ') } finally { setSaving(false) }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader><DialogTitle>إضافة عنوان جديد</DialogTitle><DialogDescription>احفظ عنوانك لتسريع الطلبات</DialogDescription></DialogHeader>
        <div className="space-y-3">
          <div><Label>اسم العنوان</Label><Input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="المنزل، العمل..." /></div>
          <div><Label>وصف العنوان (اختياري)</Label><Input value={addressText} onChange={(e) => setAddressText(e.target.value)} placeholder="الحي، الشارع..." /></div>
          <Button onClick={() => { if (!navigator.geolocation) { toast.error('المتصفح لا يدعم الموقع'); return } setLocating(true); navigator.geolocation.getCurrentPosition((p) => { setLocation({ lat: p.coords.latitude, lng: p.coords.longitude }); setLocating(false); toast.success('تم تحديد الموقع') }, () => { setLocating(false); toast.error('فشل تحديد الموقع') }, { enableHighAccuracy: true, timeout: 15000 }) }} disabled={locating} variant="outline" className="w-full rounded-lg">{locating ? <Loader2 className="w-4 h-4 ml-2 animate-spin" /> : <MapPin className="w-4 h-4 ml-2" />}{location ? 'تم تحديد الموقع ✓' : 'تحديد موقعي'}</Button>
          <Button onClick={handleSave} disabled={saving || !location} className="w-full rounded-lg bg-black hover:bg-gray-800">{saving ? <Loader2 className="w-4 h-4 ml-2 animate-spin" /> : null}حفظ العنوان</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function AddCardDialog({ open, onOpenChange, onSaved }: { open: boolean; onOpenChange: (v: boolean) => void; onSaved: () => void }) {
  const [cardNumber, setCardNumber] = useState('')
  const [cardName, setCardName] = useState('')
  const [expiry, setExpiry] = useState('')
  const [cvc, setCvc] = useState('')
  const [saveAsDefault, setSaveAsDefault] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const detectBrand = (n: string) => { const c = n.replace(/\s/g, ''); if (c.startsWith('4')) return 'Visa'; if (c.startsWith('5') || c.startsWith('2')) return 'Mastercard'; return 'Card' }
  const formatNum = (v: string) => v.replace(/\D/g, '').slice(0, 16).replace(/(\d{4})(?=\d)/g, '$1 ')
  const formatExp = (v: string) => { const c = v.replace(/\D/g, '').slice(0, 4); return c.length >= 3 ? c.slice(0, 2) + '/' + c.slice(2) : c }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    const clean = cardNumber.replace(/\s/g, '')
    if (clean.length < 16) { setError('رقم البطاقة غير مكتمل'); return }
    if (!cardName) { setError('اسم حامل البطاقة مطلوب'); return }
    if (expiry.length < 5) { setError('تاريخ الانتهاء غير صحيح'); return }
    if (cvc.length < 3) { setError('رمز CVC غير صحيح'); return }
    setSaving(true)
    try {
      const [month, year] = expiry.split('/')
      await userAPI.saveCard({ paymobToken: 'demo-' + Date.now(), brand: detectBrand(clean), last4: clean.slice(-4), expMonth: parseInt(month), expYear: 2000 + parseInt(year), cardholderName: cardName, isDefault: saveAsDefault })
      toast.success('تم حفظ البطاقة ✓'); setCardNumber(''); setCardName(''); setExpiry(''); setCvc(''); setSaveAsDefault(false); onSaved()
    } catch { setError('فشل الحفظ') } finally { setSaving(false) }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) { setCardNumber(''); setCardName(''); setExpiry(''); setCvc(''); setError('') } onOpenChange(v) }}>
      <DialogContent className="max-w-md max-h-[92dvh] overflow-y-auto rounded-lg">
        <DialogHeader><DialogTitle className="flex items-center gap-2"><CreditCard className="w-5 h-5 text-black" /> إضافة بطاقة جديدة</DialogTitle></DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="relative bg-gradient-to-br from-black to-gray-800 rounded-lg p-5 text-white overflow-hidden">
            <div className="relative">
              <div className="flex justify-between items-start mb-6"><div className="w-10 h-7 rounded bg-yellow-400/80" /><span className="text-sm font-bold">{detectBrand(cardNumber)}</span></div>
              <p className="font-mono text-lg tracking-wider mb-4" dir="ltr">{cardNumber || '•••• •••• •••• ••••'}</p>
              <div className="flex justify-between items-end"><div><p className="text-[9px] opacity-70 uppercase">حامل البطاقة</p><p className="text-sm font-medium">{cardName || 'الاسم'}</p></div><div><p className="text-[9px] opacity-70 uppercase">انتهاء</p><p className="text-sm font-mono" dir="ltr">{expiry || 'MM/YY'}</p></div></div>
            </div>
          </div>
          <div><Label>رقم البطاقة *</Label><Input value={cardNumber} onChange={(e) => setCardNumber(formatNum(e.target.value))} placeholder="0000 0000 0000 0000" dir="ltr" inputMode="numeric" maxLength={19} required className="font-mono" /></div>
          <div><Label>اسم حامل البطاقة *</Label><Input value={cardName} onChange={(e) => setCardName(e.target.value)} placeholder="AHMED MOHAMED" dir="ltr" required className="uppercase" /></div>
          <div className="grid grid-cols-2 gap-3">
            <div><Label>تاريخ الانتهاء *</Label><Input value={expiry} onChange={(e) => setExpiry(formatExp(e.target.value))} placeholder="MM/YY" dir="ltr" inputMode="numeric" maxLength={5} required className="font-mono" /></div>
            <div><Label>CVC *</Label><Input value={cvc} onChange={(e) => setCvc(e.target.value.replace(/\D/g, '').slice(0, 4))} placeholder="123" dir="ltr" inputMode="numeric" maxLength={4} required className="font-mono" /></div>
          </div>
          <label className="flex items-center gap-2 cursor-pointer p-2 rounded-lg hover:bg-gray-50"><input type="checkbox" checked={saveAsDefault} onChange={(e) => setSaveAsDefault(e.target.checked)} className="w-4 h-4 accent-black" /><span className="text-sm">تعيين كبطاقة افتراضية</span></label>
          {error && <p className="text-sm text-red-600 bg-red-50 rounded-lg p-2">{error}</p>}
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 flex items-start gap-2"><Lock className="w-4 h-4 text-gray-500 flex-shrink-0 mt-0.5" /><div><p className="text-xs font-medium text-black">دفع آمن ومشفّر</p><p className="text-[10px] text-gray-500">لا نحفظ أرقام البطاقات</p></div></div>
          <Button type="submit" disabled={saving} className="w-full h-12 rounded-lg font-bold bg-black hover:bg-gray-800">{saving ? <Loader2 className="w-4 h-4 ml-2 animate-spin" /> : null}<CreditCard className="w-4 h-4 ml-2" /> حفظ البطاقة</Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
