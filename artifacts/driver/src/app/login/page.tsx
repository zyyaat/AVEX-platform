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
      if (useAuth.getState().isAuthenticated) router.replace('/')
    })
  }, [router, initialize])

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
      <div className="px-4 h-14 flex items-center">
        <button
          onClick={() => router.back()}
          className="w-9 h-9 rounded-full hover:bg-gray-100 flex items-center justify-center transition-colors"
          aria-label="رجوع"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
      </div>

      <div className="flex-1 flex flex-col items-center justify-center px-6 -mt-10">
        <motion.div
          initial={{ opacity: 0, scale: 0.85, y: 8 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          transition={{ duration: 0.4, ease: 'easeOut' }}
          className="w-20 h-20 rounded-2xl flex items-center justify-center mb-5 shadow-lg"
          style={{ backgroundColor: '#FF6B35' }}
        >
          <Bike className="w-10 h-10 text-white" strokeWidth={2.5} />
        </motion.div>
        <motion.h1
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.05 }}
          className="text-2xl font-bold mb-1"
        >
          AVEX Driver
        </motion.h1>
        <motion.p
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="text-sm text-gray-500 mb-8 text-center"
        >
          تطبيق المندوب — للمندوبين المعتمدين
        </motion.p>

        <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-3" noValidate>
          {error && (
            <motion.div
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-red-50 border border-red-200 rounded-lg p-3 flex items-start gap-2 text-sm"
              role="alert"
            >
              <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
              <span className="text-red-700">{error}</span>
            </motion.div>
          )}

          <div className="relative">
            <Phone className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
            <input
              type="tel"
              inputMode="tel"
              dir="ltr"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="01xxxxxxxxx"
              autoComplete="tel"
              disabled={isLoading}
              className="w-full h-12 pr-10 pl-4 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500 transition-colors disabled:opacity-50"
            />
          </div>

          <div className="relative">
            <Lock className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
            <input
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="كلمة المرور"
              autoComplete="current-password"
              disabled={isLoading}
              className="w-full h-12 pr-10 pl-10 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500 transition-colors disabled:opacity-50"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700 transition-colors"
              aria-label={showPassword ? 'إخفاء' : 'إظهار'}
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full h-12 rounded-lg font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            style={{ backgroundColor: '#FF6B35' }}
          >
            {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : 'تسجيل الدخول'}
          </button>
        </form>

        <div className="mt-6 w-full max-w-sm bg-gray-50 border border-gray-200 rounded-lg p-3 flex items-start gap-2">
          <AlertCircle className="w-4 h-4 text-gray-500 flex-shrink-0 mt-0.5" />
          <p className="text-xs text-gray-600 leading-relaxed">
            هذا التطبيق للمندوبين المعتمدين فقط. تواصل مع الإدارة للحصول على حساب.
          </p>
        </div>

        <div className="mt-6 text-center text-xs text-gray-400">
          <p>حساب تجريبي:</p>
          <p dir="ltr" className="mt-1 font-mono">01012345678 / 12345678</p>
        </div>
      </div>
    </div>
  )
}
