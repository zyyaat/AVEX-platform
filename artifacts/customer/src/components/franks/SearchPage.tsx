
import { useState, useEffect, useRef } from 'react'
import { Search, X, ArrowRight, Clock, TrendingUp } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MenuCard, MenuCardSkeleton } from './MenuCard'

interface MenuItemType {
  id: string; name: string; nameAr: string; description: string; descriptionAr: string
  price: number; image: string; imageUrl: string | null; isPopular: boolean
  rating: number; ratingCount?: number; prepTime: number; calories: number
  category?: { nameAr: string; icon: string }
}
interface Category { id: string; name: string; nameAr: string; icon: string; items: MenuItemType[] }

interface SearchPageProps { onBack: () => void }

const FILTERS = [
  { id: 'all', label: 'الجميع' },
  { id: 'fast', label: 'توصيل سريع' },
  { id: 'offers', label: 'عروض' },
  { id: 'popular', label: 'الأكثر طلباً' },
]

export function SearchPage({ onBack }: SearchPageProps) {
  const [query, setQuery] = useState('')
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [allItems, setAllItems] = useState<MenuItemType[]>([])
  const [activeFilter, setActiveFilter] = useState('all')
  const [recentSearches, setRecentSearches] = useState<string[]>([])
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    fetch('/api/v1/categories').then(r => r.json()).then(data => {
      setCategories(data.categories || [])
      const items = (data.categories || []).flatMap((c: Category) =>
        (c.items || []).map((i: MenuItemType) => ({ ...i, category: { nameAr: c.nameAr, icon: c.icon } }))
      )
      setAllItems(items)
      setLoading(false)
    }).catch(() => setLoading(false))

    // Load recent searches from localStorage
    try {
      const saved = localStorage.getItem('avex-recent-searches')
      if (saved) {
        const parsed = JSON.parse(saved)
        setTimeout(() => setRecentSearches(parsed), 0)
      }
    } catch {}
    setTimeout(() => inputRef.current?.focus(), 100)
  }, [])

  const saveSearch = (q: string) => {
    if (!q.trim()) return
    const updated = [q, ...recentSearches.filter(s => s !== q)].slice(0, 6)
    setRecentSearches(updated)
    localStorage.setItem('avex-recent-searches', JSON.stringify(updated))
  }

  const clearRecent = () => {
    setRecentSearches([])
    localStorage.removeItem('avex-recent-searches')
  }

  // Filter logic
  let results = query.trim() === '' ? [] : allItems.filter(i =>
    i.nameAr.includes(query) ||
    i.name.toLowerCase().includes(query.toLowerCase()) ||
    i.descriptionAr.includes(query) ||
    (i.category?.nameAr || '').includes(query)
  )

  // Apply filters
  if (activeFilter === 'fast') results = results.filter(i => i.prepTime <= 15)
  if (activeFilter === 'offers') results = results.filter(i => i.isPopular)
  if (activeFilter === 'popular') results = [...results].sort((a, b) => b.rating - a.rating)

  const popularItems = allItems.filter(i => i.isPopular).slice(0, 6)
  const showEmpty = query.trim() !== '' && results.length === 0

  return (
    <div className="min-h-dvh bg-white pb-14 sm:pb-0" dir="rtl">
      {/* Search header */}
      <div className="sticky top-0 z-20 bg-white border-b border-gray-100 px-4 py-3 flex items-center gap-2">
        <button onClick={onBack} className="w-9 h-9 flex items-center justify-center rounded-lg hover:bg-gray-50">
          <ArrowRight className="w-5 h-5 text-black" />
        </button>
        <div className="flex-1 relative">
          <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && saveSearch(query)}
            placeholder="ابحث عن أطباق، مطاعم..."
            className="pr-9 h-10 rounded-lg bg-gray-50 border-gray-200 focus:bg-white text-sm"
          />
          {query && (
            <button onClick={() => setQuery('')} className="absolute left-2 top-1/2 -translate-y-1/2 w-6 h-6 flex items-center justify-center rounded-full hover:bg-gray-100">
              <X className="w-3.5 h-3.5 text-gray-400" />
            </button>
          )}
        </div>
      </div>

      {/* Filter tabs */}
      {query.trim() !== '' && (
        <div className="sticky top-[57px] z-10 bg-white border-b border-gray-100 px-4 py-2 flex gap-2 overflow-x-auto scrollbar-hide">
          {FILTERS.map(f => (
            <button
              key={f.id}
              onClick={() => setActiveFilter(f.id)}
              className={`flex-shrink-0 px-3 py-1.5 rounded-md text-xs font-medium transition-fluent ${
                activeFilter === f.id ? 'bg-black text-white' : 'bg-gray-50 text-gray-600 hover:bg-gray-100'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      )}

      {/* Content */}
      <div className="container mx-auto px-4 py-4 max-w-2xl">
        {loading ? (
          <div className="grid grid-cols-2 gap-3">
            {Array.from({ length: 4 }).map((_, i) => <MenuCardSkeleton key={i} />)}
          </div>
        ) : query.trim() === '' ? (
          /* Empty state */
          <div className="space-y-6">
            {/* Recent searches */}
            {recentSearches.length > 0 && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-bold text-black flex items-center gap-1.5">
                    <Clock className="w-4 h-4" /> آخر البحث
                  </h3>
                  <button onClick={clearRecent} className="text-xs text-gray-400 hover:text-black">مسح</button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {recentSearches.map((s, i) => (
                    <button
                      key={i}
                      onClick={() => setQuery(s)}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-gray-50 border border-gray-200 text-sm text-gray-700 hover:border-gray-300"
                    >
                      {s}
                      <ArrowRight className="w-3 h-3 text-gray-400" />
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Popular */}
            <div>
              <h3 className="text-sm font-bold text-black mb-3 flex items-center gap-1.5">
                <TrendingUp className="w-4 h-4" /> الأكثر طلباً
              </h3>
              <div className="grid grid-cols-2 gap-3">
                {popularItems.map(item => <MenuCard key={item.id} item={item} />)}
              </div>
            </div>

            {/* Categories */}
            <div>
              <h3 className="text-sm font-bold text-black mb-3">التصنيفات</h3>
              <div className="flex flex-wrap gap-2">
                {categories.map(cat => (
                  <button
                    key={cat.id}
                    onClick={() => { setQuery(cat.nameAr); saveSearch(cat.nameAr) }}
                    className="px-3 py-2 rounded-lg bg-gray-50 border border-gray-200 text-sm font-medium text-gray-700 hover:border-gray-300"
                  >
                    {cat.icon} {cat.nameAr}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : results.length > 0 ? (
          /* Results */
          <div>
            <p className="text-xs text-gray-400 mb-3">{results.length} نتيجة</p>
            <div className="grid grid-cols-2 gap-3">
              {results.map(item => <MenuCard key={item.id} item={item} />)}
            </div>
          </div>
        ) : (
          /* No results */
          <div className="text-center py-16">
            <div className="w-14 h-14 rounded-full bg-gray-50 flex items-center justify-center mx-auto mb-3">
              <Search className="w-6 h-6 text-gray-300" />
            </div>
            <p className="text-sm font-medium text-black">لا توجد نتائج لـ &quot;{query}&quot;</p>
            <p className="text-xs text-gray-400 mt-1">جرّب كلمات بحث أخرى</p>
          </div>
        )}
      </div>
    </div>
  )
}
