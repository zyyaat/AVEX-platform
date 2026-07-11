import { useState } from 'react'
import { motion } from 'framer-motion'
import {
  Bike, Phone, Lock, Loader2, Eye, EyeOff, AlertCircle,
} from 'lucide-react'
import { useAuth } from '@/store/auth'
import { toast } from 'sonner'

export default function LoginPage() {
  const { login, isAuthenticated } = useAuth()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!phone || !password) {
      setError('ادخل رقم الهاتف وكلمة المرور')
      return
    }
    setLoading(true)
    try {
      await login(phone, password)
      toast.success('تم تسجيل الدخول بنجاح')
    } catch (err: any) {
      setError(err.message || 'فشل تسجيل الدخول')
    } finally {
      setLoading(false)
    }
  }

  // If already authenticated, this page won't be shown (App.tsx handles routing)
  if (isAuthenticated) return null

  return (
    <div className="min-h-dvh bg-white flex flex-col" dir="rtl">
      <div className="flex-1 flex flex-col items-center justify-center px-6">
        <motion.div
          initial={{ opacity: 0, scale: 0.85 }}
          animate={{ opacity: 1, scale: 1 }}
          className="w-20 h-20 rounded-2xl flex items-center justify-center mb-5 shadow-lg"
          style={{ backgroundColor: '#FF6B35' }}
        >
          <Bike className="w-10 h-10 text-white" strokeWidth={2.5} />
        </motion.div>
        <h1 className="text-2xl font-bold mb-1">AVEX Driver</h1>
        <p className="text-sm text-gray-500 mb-8 text-center">تطبيق المندوب — للمندوبين المعتمدين</p>

        <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-3" noValidate>
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-3 flex items-start gap-2 text-sm">
              <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
              <span className="text-red-700">{error}</span>
            </div>
          )}
          <div className="relative">
            <Phone className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
            <input
              type="tel"
              dir="ltr"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="01xxxxxxxxx"
              autoComplete="tel"
              disabled={loading}
              className="w-full h-12 pr-10 pl-4 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500"
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
              disabled={loading}
              className="w-full h-12 pr-10 pl-10 rounded-lg border border-gray-200 bg-white text-right focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700"
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full h-12 rounded-lg font-medium text-white transition-colors flex items-center justify-center gap-2"
            style={{ backgroundColor: '#FF6B35' }}
          >
            {loading ? <Loader2 className="w-5 h-5 animate-spin" /> : 'تسجيل الدخول'}
          </button>
        </form>

        <div className="mt-6 text-center text-xs text-gray-400">
          <p>حساب تجريبي:</p>
          <p dir="ltr" className="mt-1 font-mono">01012345678 / 12345678</p>
        </div>
      </div>
    </div>
  )
}
