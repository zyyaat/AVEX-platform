
import { useState, useEffect, useRef } from 'react'
import { Loader2, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MenuCard, MenuCardSkeleton } from './MenuCard'
import { ProductDetail } from './ProductDetail'

interface MenuItemType {
  id: string; name: string; nameAr: string; description: string; descriptionAr: string
  price: number; image: string; imageUrl: string | null; isPopular: boolean
  rating: number; ratingCount?: number; prepTime: number; calories: number
}
interface Category { id: string; name: string; nameAr: string; icon: string; imageUrl: string | null; items: MenuItemType[] }

export function MenuSection() {
  const [categories, setCategories] = useState<Category[]>([])
  const [activeCategory, setActiveCategory] = useState('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [selectedItem, setSelectedItem] = useState<MenuItemType | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const sectionRefs = useRef<Record<string, HTMLDivElement | null>>({})

  useEffect(() => {
    fetch('/api/v1/categories').then(r => r.json()).then(data => { setCategories(data.categories || []); setLoading(false) }).catch(() => setLoading(false))
  }, [])

  const filtered = categories
    .map((cat) => ({
      ...cat,
      items: cat.items.filter(
        (i) =>
          (activeCategory === 'all' || cat.id === activeCategory) &&
          (searchQuery === '' ||
            i.nameAr.includes(searchQuery) ||
            i.name.toLowerCase().includes(searchQuery.toLowerCase()))
      ),
    }))
    .filter((c) => c.items.length > 0)

  const popular = categories.flatMap(c => c.items).filter(i => i.isPopular).slice(0, 8)

  const scrollTo = (id: string) => {
    setActiveCategory(id)
    if (id === 'all') document.getElementById('menu-start')?.scrollIntoView({ behavior: 'smooth' })
    else sectionRefs.current[id]?.scrollIntoView({ behavior: 'smooth' })
  }

  const openDetail = (item: MenuItemType) => {
    setSelectedItem(item)
    setDetailOpen(true)
  }

  return (
    <section id="menu" className="py-8 md:py-10 bg-white">
      <div className="container mx-auto px-4">
        {/* Search */}
        <div className="max-w-md mx-auto mb-6 relative">
          <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input value={searchQuery} onChange={e => setSearchQuery(e.target.value)} placeholder="ابحث عن أطباقك المفضلة..." className="pr-9 h-10 rounded-lg bg-gray-50 border-gray-200 focus:bg-white" />
        </div>

        {/* Categories */}
        <div className="mb-8 -mx-4 px-4">
          <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
            <button onClick={() => scrollTo('all')} className={`flex-shrink-0 px-3 py-1.5 rounded-md text-sm font-medium transition-fluent ${activeCategory === 'all' ? 'bg-black text-white' : 'bg-gray-50 text-gray-600 hover:bg-gray-100'}`}>الكل</button>
            {categories.map(cat => (
              <button key={cat.id} onClick={() => scrollTo(cat.id)} className={`flex-shrink-0 px-3 py-1.5 rounded-md text-sm font-medium transition-fluent ${activeCategory === cat.id ? 'bg-black text-white' : 'bg-gray-50 text-gray-600 hover:bg-gray-100'}`}>
                {cat.icon} {cat.nameAr}
              </button>
            ))}
          </div>
        </div>

        <div id="menu-start" />

        {/* Loading skeleton */}
        {loading ? (
          <div className="space-y-8">
            <div>
              <div className="h-5 w-24 bg-gray-100 rounded animate-pulse mb-3" />
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                {Array.from({ length: 4 }).map((_, i) => <MenuCardSkeleton key={i} />)}
              </div>
            </div>
            <div>
              <div className="h-5 w-24 bg-gray-100 rounded animate-pulse mb-3" />
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                {Array.from({ length: 4 }).map((_, i) => <MenuCardSkeleton key={i} />)}
              </div>
            </div>
          </div>
        ) : (
          <>
            {/* Popular */}
            {!searchQuery && activeCategory === 'all' && popular.length > 0 && (
              <div className="mb-10">
                <h3 className="text-base font-bold text-black mb-3">الأكثر طلباً</h3>
                <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-hide -mx-4 px-4">
                  {popular.map(item => <div key={item.id} className="w-56 flex-shrink-0"><MenuCard item={item} onClick={() => openDetail(item)} /></div>)}
                </div>
              </div>
            )}

            {/* Menu by category */}
            <div className="space-y-8">
              {filtered.map(cat => (
                <div key={cat.id} ref={el => { sectionRefs.current[cat.id] = el }} className="scroll-mt-20">
                  <div className="flex items-center gap-2 mb-3">
                    <h3 className="text-lg font-bold text-black">{cat.nameAr}</h3>
                    <span className="text-xs text-gray-400">{cat.items.length}</span>
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    {cat.items.map(item => <MenuCard key={item.id} item={item} onClick={() => openDetail(item)} />)}
                  </div>
                </div>
              ))}
            </div>

            {/* Empty state */}
            {filtered.length === 0 && (
              <div className="text-center py-16">
                <div className="w-14 h-14 rounded-full bg-gray-50 flex items-center justify-center mx-auto mb-3">
                  <Search className="w-6 h-6 text-gray-300" />
                </div>
                <p className="text-sm font-medium text-black">لا توجد نتائج</p>
                <p className="text-xs text-gray-400 mt-1">جرّب كلمات بحث أخرى</p>
              </div>
            )}
          </>
        )}
      </div>

      {/* Product Detail Bottom Sheet */}
      <ProductDetail item={selectedItem} open={detailOpen} onOpenChange={setDetailOpen} />
    </section>
  )
}
