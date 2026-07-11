
import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Loader2, User, Phone, Lock, Mail, CheckCircle2, UserPlus, LogIn } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
} from '@/components/ui/dialog'
import { useAuth } from '@/store/auth'
import { toast } from 'sonner'

interface AuthDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialMode?: 'login' | 'register'
}

export function AuthDialog({ open, onOpenChange, initialMode = 'login' }: AuthDialogProps) {
  const [mode, setMode] = useState<'login' | 'register'>(initialMode)
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState({ name: '', phone: '', password: '', email: '' })
  const { login, register } = useAuth()

  useEffect(() => {
    if (open) setMode(initialMode)
  }, [open, initialMode])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Normalize phone (handle +20)
    let normalizedPhone = form.phone.replace(/[^\d+]/g, '')
    if (normalizedPhone.startsWith('+20')) {
      normalizedPhone = '0' + normalizedPhone.slice(3)
    } else if (normalizedPhone.startsWith('20') && normalizedPhone.length === 13) {
      normalizedPhone = '0' + normalizedPhone.slice(2)
    }

    // Validate Egyptian phone: 010/011/012/015 + 8 digits = 11 total
    if (!/^01[0125][0-9]{8}$/.test(normalizedPhone)) {
      toast.error('رقم الهاتف يجب أن يكون 11 رقماً مصرياً ويبدأ بـ 010 أو 011 أو 012 أو 015')
      return
    }

    if (form.password.length < 6) {
      toast.error('كلمة المرور يجب أن تكون 6 أحرف على الأقل')
      return
    }

    if (mode === 'register' && form.name.length < 2) {
      toast.error('الاسم يجب أن يكون حرفين على الأقل')
      return
    }

    setLoading(true)
    try {
      if (mode === 'login') {
        await login(normalizedPhone, form.password)
        toast.success('تم تسجيل الدخول بنجاح! 🎉')
      } else {
        await register(form.name, normalizedPhone, form.password, form.email || undefined)
        toast.success('تم إنشاء حسابك بنجاح! مرحباً بك في AVEX 🚀')
      }
      setForm({ name: '', phone: '', password: '', email: '' })
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'فشل العملية')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md max-h-[92dvh] overflow-y-auto overscroll-contain rounded-2xl">
        <DialogHeader>
          <DialogTitle className="text-xl flex items-center gap-2">
            {mode === 'login' ? <><LogIn className="w-5 h-5 text-black" /> تسجيل الدخول</> : <><UserPlus className="w-5 h-5 text-black" /> إنشاء حساب جديد</>}
          </DialogTitle>
          <DialogDescription>
            {mode === 'login' ? 'سجّل دخولك للوصول لطلباتك ومتابعتها بسهولة' : 'أنشئ حساباً للاستفادة من كل المزايا'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex gap-1 bg-gray-100 p-1 rounded-lg">
          <button type="button" onClick={() => setMode('login')} className={`flex-1 py-2 rounded-lg text-sm font-bold transition-all ${mode === 'login' ? 'bg-white shadow-sm text-black' : 'text-gray-500'}`}>دخول</button>
          <button type="button" onClick={() => setMode('register')} className={`flex-1 py-2 rounded-lg text-sm font-bold transition-all ${mode === 'register' ? 'bg-white shadow-sm text-black' : 'text-gray-500'}`}>حساب جديد</button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3" noValidate>
          <AnimatePresence mode="wait">
            {mode === 'register' && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }} className="space-y-1.5">
                <Label className="text-sm font-medium flex items-center gap-1.5"><User className="w-3.5 h-3.5" /> الاسم الكامل <span className="text-red-600">*</span></Label>
                <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="أحمد محمد" required className="rounded-lg" />
              </motion.div>
            )}
          </AnimatePresence>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium flex items-center gap-1.5"><Phone className="w-3.5 h-3.5" /> رقم الهاتف <span className="text-red-600">*</span> <span className="text-[10px] text-gray-500 mr-auto">11 رقم</span></Label>
            <Input
              type="tel"
              value={form.phone}
              onChange={(e) => {
                let val = e.target.value.replace(/[^\d+]/g, '')
                if (val.startsWith('+20')) { val = '0' + val.slice(3) }
                else if (val.startsWith('20') && val.length === 13 && val[2] === '1') { val = '0' + val.slice(2) }
                else if (val.startsWith('+')) { val = val.replace(/\+/g, '') }
                val = val.replace(/[^0-9]/g, '').slice(0, 11)
                setForm({ ...form, phone: val })
              }}
              placeholder="01012345678"
              required
              autoComplete="off"
              name="phone"
              className={`rounded-lg ${form.phone && !/^01[0125][0-9]{8}$/.test(form.phone) ? 'border-red-500' : form.phone.length === 11 && /^01[0125][0-9]{8}$/.test(form.phone) ? 'border-black' : ''}`}
              dir="ltr"
              inputMode="tel"
            />
            {form.phone && form.phone.length === 11 && /^01[0125][0-9]{8}$/.test(form.phone) && (
              <p className="text-[10px] text-gray-500 flex items-center gap-1"><CheckCircle2 className="w-3 h-3" /> رقم مصري صحيح</p>
            )}
          </div>

          <AnimatePresence mode="wait">
            {mode === 'register' && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }} className="space-y-1.5">
                <Label className="text-sm font-medium flex items-center gap-1.5"><Mail className="w-3.5 h-3.5" /> البريد الإلكتروني (اختياري)</Label>
                <Input type="text" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="ahmed@example.com" className="rounded-lg" dir="ltr" autoComplete="off" name="email" inputMode="email" />
              </motion.div>
            )}
          </AnimatePresence>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium flex items-center gap-1.5"><Lock className="w-3.5 h-3.5" /> كلمة المرور <span className="text-red-600">*</span></Label>
            <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="••••••" required className="rounded-lg" dir="ltr" minLength={6} />
            <p className="text-[10px] text-gray-500">6 أحرف على الأقل</p>
          </div>

          {mode === 'register' && (
            <div className="bg-gray-50 border border-gray-100 rounded-xl p-3 space-y-1.5">
              <p className="text-xs font-bold text-black mb-2">مزايا إنشاء حساب:</p>
              {['📦 تتبع جميع طلباتك من مكان واحد', '📍 حفظ عناوينك لإعادة الاستخدام', '❤️ حفظ منتجاتك المفضلة', '🎁 نقاط ولاء مع كل طلب'].map((b, i) => <p key={i} className="text-[11px] text-gray-500">{b}</p>)}
            </div>
          )}

          <Button type="submit" disabled={loading} className="w-full h-12 text-base font-bold rounded-xl shadow-lg bg-black hover:bg-gray-800">
            {loading ? <><Loader2 className="w-5 h-5 ml-2 animate-spin" /> جاري المعالجة...</> : mode === 'login' ? 'تسجيل الدخول' : 'إنشاء الحساب'}
          </Button>

          <p className="text-center text-xs text-gray-500">
            {mode === 'login' ? <>ليس لديك حساب؟ <button type="button" onClick={() => setMode('register')} className="text-black font-bold hover:underline">أنشئ حساباً</button></> : <>لديك حساب بالفعل؟ <button type="button" onClick={() => setMode('login')} className="text-black font-bold hover:underline">سجّل دخولك</button></>}
          </p>
        </form>
      </DialogContent>
    </Dialog>
  )
}
