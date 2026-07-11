import { useState, useEffect } from 'react'
import { useRouter, usePathname, Link } from '@/lib/navigation'
import {
  Bike, Home, Package, Wallet, Clock, User, LifeBuoy, LogOut, Menu, X,
} from 'lucide-react'
import { useAuth } from '@/store/auth'

const navItems = [
  { href: '/', label: 'الرئيسية', icon: Home },
  { href: '/earnings', label: 'الأرباح', icon: Wallet },
  { href: '/history', label: 'السجل', icon: Clock },
  { href: '/profile', label: 'الملف الشخصي', icon: User },
  { href: '/support', label: 'الدعم', icon: LifeBuoy },
]

export function DriverLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, userID, logout, initialize } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [bootChecked, setBootChecked] = useState(false)

  useEffect(() => {
    initialize().then(() => setBootChecked(true))
  }, [initialize])

  useEffect(() => {
    if (bootChecked && !isAuthenticated && pathname !== '/login') {
      router.replace('/login')
    }
  }, [bootChecked, isAuthenticated, pathname, router])

  if (pathname === '/login') return <>{children}</>
  if (!bootChecked) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-white">
        <div className="animate-pulse text-gray-400">جاري التحميل...</div>
      </div>
    )
  }
  if (!isAuthenticated) return null

  return (
    <div className="min-h-dvh bg-gray-50 flex" dir="rtl">
      {/* Sidebar (desktop) */}
      <aside className="hidden md:flex md:flex-col md:w-64 md:fixed md:inset-y-0 md:right-0 bg-white border-l border-gray-200">
        <div className="h-14 px-4 flex items-center border-b border-gray-200">
          <Bike className="w-6 h-6 text-orange-500 ml-2" />
          <span className="font-bold text-lg">AVEX Driver</span>
        </div>
        <nav className="flex-1 overflow-y-auto py-2">
          {navItems.map((item) => {
            const Icon = item.icon
            const active = pathname === item.href
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 px-4 py-2.5 text-sm transition-colors ${
                  active ? 'bg-orange-500 text-white font-bold' : 'text-gray-600 hover:bg-gray-100'
                }`}
              >
                <Icon className="w-4 h-4" />
                {item.label}
              </Link>
            )
          })}
        </nav>
        <div className="border-t border-gray-200 p-3">
          <div className="text-xs text-gray-500 mb-2 px-2">المندوب: {userID?.slice(0, 8) || '—'}</div>
          <button
            onClick={() => { logout(); router.replace('/login') }}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <LogOut className="w-4 h-4" /> تسجيل الخروج
          </button>
        </div>
      </aside>

      {/* Mobile sidebar */}
      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-40 bg-black/50" onClick={() => setSidebarOpen(false)}>
          <aside className="absolute right-0 top-0 bottom-0 w-72 bg-white flex flex-col" onClick={(e) => e.stopPropagation()}>
            <div className="h-14 px-4 flex items-center justify-between border-b border-gray-200">
              <span className="font-bold">🚴 AVEX Driver</span>
              <button onClick={() => setSidebarOpen(false)} className="w-8 h-8 rounded-full hover:bg-gray-100 flex items-center justify-center">
                <X className="w-5 h-5" />
              </button>
            </div>
            <nav className="flex-1 overflow-y-auto py-2">
              {navItems.map((item) => {
                const Icon = item.icon
                const active = pathname === item.href
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setSidebarOpen(false)}
                    className={`flex items-center gap-3 px-4 py-3 text-sm transition-colors ${
                      active ? 'bg-orange-500 text-white font-bold' : 'text-gray-600 hover:bg-gray-100'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                )
              })}
            </nav>
            <button
              onClick={() => { logout(); router.replace('/login') }}
              className="m-3 flex items-center gap-2 px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg"
            >
              <LogOut className="w-4 h-4" /> تسجيل الخروج
            </button>
          </aside>
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 md:mr-64">
        {/* Topbar */}
        <header className="sticky top-0 z-30 bg-white border-b border-gray-200 h-14 flex items-center justify-between px-4">
          <button onClick={() => setSidebarOpen(true)} className="md:hidden w-9 h-9 rounded-full hover:bg-gray-100 flex items-center justify-center">
            <Menu className="w-5 h-5" />
          </button>
          <div className="hidden md:block text-sm text-gray-500">تطبيق المندوب</div>
          <div className="md:hidden font-bold">🚴 AVEX</div>
        </header>

        <main className="p-4 md:p-6 max-w-4xl mx-auto">{children}</main>
      </div>
    </div>
  )
}
