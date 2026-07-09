import { motion, AnimatePresence } from 'framer-motion'
import {
  X, Clock, MessageSquare, FileText, Wallet,
  User, Camera, Settings, Globe, LogOut,
} from 'lucide-react'
import { useRouter } from '@/lib/navigation'
import { useAuth } from '@/store/auth'
import { useDriver } from '@/store/driver'
import { toast } from 'sonner'

interface SideDrawerProps {
  open: boolean
  onClose: () => void
}

export function SideDrawer({ open, onClose }: SideDrawerProps) {
  const router = useRouter()
  const { logout, userID } = useAuth()
  const { driver } = useDriver()

  const handleNavigate = (path: string) => {
    onClose()
    router.push(path)
  }

  const handleLogout = () => {
    logout()
    useDriver.getState().clear()
    toast.success('تم تسجيل الخروج')
    router.replace('/login')
  }

  const menuItems = [
    { icon: Clock, label: 'السجل', path: '/history' },
    { icon: MessageSquare, label: 'الرسائل', path: '/support', badge: 0 },
    { icon: FileText, label: 'المستندات', path: '/profile' },
    { icon: Wallet, label: 'المدفوعات', path: '/earnings' },
    { icon: User, label: 'تعديل الملف الشخصي', path: '/profile' },
    { icon: Camera, label: 'صورة', path: '/profile' },
    { icon: Settings, label: 'الإعدادات', path: '/profile' },
    { icon: Globe, label: 'اللغة', path: '/profile' },
  ]

  return (
    <AnimatePresence>
      {open && (
        <>
          {/* Overlay */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-black/40 z-40"
          />

          {/* Drawer — opens from the RIGHT (RTL) */}
          <motion.div
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 25, stiffness: 200 }}
            className="fixed top-0 right-0 bottom-0 w-80 max-w-[85vw] bg-white z-50 flex flex-col"
            style={{ paddingTop: 'env(safe-area-inset-top, 0px)' }}
          >
            {/* Header */}
            <div
              className="px-5 py-6 flex flex-col"
              style={{ backgroundColor: '#FF6B35' }}
            >
              <button
                onClick={onClose}
                className="absolute top-4 left-4 w-8 h-8 rounded-full bg-white/20 flex items-center justify-center text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="w-16 h-16 rounded-full bg-white/20 flex items-center justify-center mb-3">
                <User className="w-8 h-8 text-white" />
              </div>
              <h2 className="text-white font-bold text-lg">المندوب</h2>
              <p className="text-white/80 text-sm">#{userID?.slice(0, 8) || '---'}</p>
              {driver && (
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-white/90 text-xs">
                    {driver.vehicle_type === 'bike' ? 'دراجة' : driver.vehicle_type === 'car' ? 'سيارة' : 'سكوتر'}
                  </span>
                  <span className="text-white/60 text-xs">•</span>
                  <span className="text-white/90 text-xs">{driver.license_plate}</span>
                </div>
              )}
            </div>

            {/* Quick actions grid */}
            <div className="grid grid-cols-4 gap-3 p-4 border-b border-gray-100">
              {menuItems.slice(0, 4).map((item) => (
                <button
                  key={item.label}
                  onClick={() => handleNavigate(item.path)}
                  className="flex flex-col items-center gap-1.5 py-2"
                >
                  <div className="w-10 h-10 rounded-xl bg-gray-50 flex items-center justify-center">
                    <item.icon className="w-5 h-5 text-gray-700" />
                  </div>
                  <span className="text-xs text-gray-600">{item.label}</span>
                </button>
              ))}
            </div>

            {/* Menu items */}
            <div className="flex-1 overflow-y-auto py-2">
              {menuItems.slice(4).map((item) => (
                <button
                  key={item.label}
                  onClick={() => handleNavigate(item.path)}
                  className="w-full flex items-center gap-3 px-5 py-3 hover:bg-gray-50 transition-colors"
                >
                  <item.icon className="w-5 h-5 text-gray-500" />
                  <span className="text-sm text-gray-700">{item.label}</span>
                </button>
              ))}
            </div>

            {/* Logout */}
            <div className="border-t border-gray-100 p-4">
              <button
                onClick={handleLogout}
                className="w-full flex items-center gap-3 px-2 py-2 text-red-500 hover:bg-red-50 rounded-lg transition-colors"
              >
                <LogOut className="w-5 h-5" />
                <span className="text-sm font-medium">تسجيل الخروج</span>
              </button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
