import { useState, useEffect } from 'react'
import { useRouter } from '@/lib/navigation'
import { motion } from 'framer-motion'
import {
  Bike, Phone, Lock, ArrowLeft, Loader2, Eye, EyeOff,
  AlertCircle,
} from 'lucide-react'
import { useAuth } from '@/store/auth'
import { toast } from 'sonner'

export default function LoginPage() {
  const router = useRouter()
  const { login, isAuthenticated, isInitialized, isLoading, initialize } = useAuth()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    initialize().then(() => {
      // Only redirect if initialize confirmed the user is authenticated.
      const s = useAuth.getState()
      if (s.isInitialized && s.isAuthenticated) {
        router.replace('/')
      }
    })
  }, [router, initialize])

  // Also handle the case where login() sets isAuthenticated=true directly.
  useEffect(() => {
    if (isInitialized && isAuthenticated) {
      router.replace('/')
    }
  }, [isInitialized, isAuthenticated, router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!phone || !password) {
      setError('ادخل رقم الهاتف وكلمة المرور')
      return
    }
    try {
      await login(phone, password)
      toast.success('تم تسجيل الدخول بنجاح')
      router.replace('/')
    } catch (err: any) {
      setError(err.message || 'فشل تسجيل الدخول')
    }
  }

  return (
    <div className="min-h-dvh bg-white flex flex-col" dir="rtl">
      {/* Top bar */}
      <div className="px-4 h-14 flex items-center">
        <button
          onClick={() => window.history.back()}
          className="w-9 h-9 rounded-full hover:bg-gray-100 flex items-center justify-center transition-colors"
          aria-label="رجوع"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
      </div>

      {/* Logo */}
      <div className="flex-1 flex flex-col items-center justify-center px-6 pb-8">
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ type: 'spring', stiffness: 200, damping: 20 }}
          className="w-20 h-20 rounded-3xl flex items-center justify-center mb-4"
          style={{ backgroundColor: '#FF6B35' }}
        >
          <Bike className="w-10 h-10 text-white" strokeWidth={2.5} />
        </motion.div>
        <h1 className="text-2xl font-bold text-gray-900 mb-1">AVEX</h1>
        <p className="text-sm text-gray-500 mb-8">تطبيق المندوب</p>

        {/* Form */}
        <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-xl p-3 flex items-center gap-2 text-red-700 text-sm">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Phone */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-gray-700">رقم الهاتف</label>
            <div className="relative">
              <Phone className="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="01xxxxxxxxx"
                className="w-full h-12 pr-11 pl-4 rounded-xl border border-gray-200 bg-gray-50 focus:bg-white focus:border-gray-400 focus:ring-2 focus:ring-gray-100 outline-none transition-all text-gray-900 placeholder:text-gray-400"
                dir="ltr"
                autoComplete="tel"
                disabled={isLoading}
              />
            </div>
          </div>

          {/* Password */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-gray-700">كلمة المرور</label>
            <div className="relative">
              <Lock className="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full h-12 pr-11 pl-11 rounded-xl border border-gray-200 bg-gray-50 focus:bg-white focus:border-gray-400 focus:ring-2 focus:ring-gray-100 outline-none transition-all text-gray-900 placeholder:text-gray-400"
                autoComplete="current-password"
                disabled={isLoading}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              >
                {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
          </div>

          {/* Submit */}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full h-12 rounded-xl font-semibold text-white flex items-center justify-center gap-2 transition-all disabled:opacity-50 active:scale-[0.98]"
            style={{ backgroundColor: '#FF6B35' }}
          >
            {isLoading ? (
              <Loader2 className="w-5 h-5 animate-spin" />
            ) : (
              'تسجيل الدخول'
            )}
          </button>
        </form>

        <p className="text-xs text-gray-400 mt-8 text-center">
          بتسجيل الدخول، أنت توافق على الشروط والأحكام
        </p>
      </div>
    </div>
  )
}
