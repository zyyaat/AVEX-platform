
import { useState, useEffect } from 'react'
import { ArrowRight, Star, Clock, Truck, Loader2, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MenuCard, MenuCardSkeleton } from './MenuCard'
import { ProductDetail } from './ProductDetail'

interface MenuItemType {
  id: string; name: string; nameAr: string; description: string; descriptionAr: string
  price: number; image: string; imageUrl: string | null; isPopular: boolean
  rating: number; ratingCount?: number; prepTime: number; calories: number; categoryId: string
}
interface RestaurantData {
  id: string; name: string; nameAr: string; descriptionAr: string
  imageUrl: string; coverUrl: string; rating: number; ratingCount: number
  deliveryTimeMin: number; deliveryTimeMax: number; deliveryFee: number
  minOrder: number; isPro: boolean; cuisines: string; menu: MenuItemType[]
}

interface RestaurantPageProps {
  restaurantId: string
  onBack: () => void
}

export function RestaurantPage({ restaurantId, onBack }: RestaurantPageProps) {
  const [restaurant, setRestaurant] = useState<RestaurantData | null>(null)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selectedItem, setSelectedItem] = useState<MenuItemType | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  useEffect(() => {
    fetch(`/api/v1/restaurants/${restaurantId}`)
      .then(r => r.json())
      .then(data => { setRestaurant(data); setLoading(false) })
      .catch(() => setLoading(false))
  }, [restaurantId])

  const openDetail = (item: MenuItemType) => {
    setSelectedItem(item)
    setDetailOpen(true)
  }

  if (loading) {
    return (
      <div className="min-h-dvh bg-white" dir="rtl">
        <div className="h-48 bg-gray-100 animate-pulse" />
        <div className="container mx-auto px-4 py-4">
          <div className="h-6 w-40 bg-gray-100 rounded animate-pulse mb-2" />
          <div className="h-4 w-60 bg-gray-100 rounded animate-pulse mb-4" />
          <div className="grid grid-cols-2 gap-3">
            {Array.from({ length: 4 }).map((_, i) => <MenuCardSkeleton key={i} />)}
          </div>
        </div>
      </div>
    )
  }

  if (!restaurant) {
    return (
      <div className="min-h-dvh bg-white flex items-center justify-center" dir="rtl">
        <p className="text-gray-400">المطعم غير موجود</p>
      </div>
    )
  }

  const filteredMenu = search
    ? (restaurant.menu || []).filter(i => i.nameAr.includes(search) || i.name.toLowerCase().includes(search.toLowerCase()))
    : (restaurant.menu || [])

  // Group by popular first, then all
  const popular = filteredMenu.filter(i => i.isPopular)
  const rest = filteredMenu.filter(i => !i.isPopular)

  return (
    <div className="min-h-dvh bg-white pb-14 sm:pb-0" dir="rtl">
      {/* Cover */}
      <div className="relative h-40 bg-gray-100">
        {restaurant.coverUrl && <img src={restaurant.coverUrl} alt="" className="w-full h-full object-cover" />}
        <button onClick={onBack} className="absolute top-3 right-3 w-9 h-9 rounded-full bg-white/95 backdrop-blur-sm flex items-center justify-center shadow-fluent">
          <ArrowRight className="w-5 h-5 text-black" />
        </button>
      </div>

      {/* Restaurant info */}
      <div className="container mx-auto px-4 -mt-6 relative">
        <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-fluent">
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-bold text-black">{restaurant.nameAr}</h1>
                {restaurant.isPro && <span className="text-[10px] bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded font-medium">PRO</span>}
              </div>
              {restaurant.cuisines && <p className="text-xs text-gray-400 mt-0.5">{restaurant.cuisines}</p>}
              {restaurant.descriptionAr && <p className="text-sm text-gray-500 mt-1">{restaurant.descriptionAr}</p>}
            </div>
            <div className="flex items-center gap-1 bg-gray-50 rounded-md px-2 py-1 flex-shrink-0">
              <Star className="w-4 h-4 fill-black text-black" />
              <span className="text-sm font-bold text-black">{restaurant.rating}</span>
              <span className="text-xs text-gray-400">({restaurant.ratingCount})</span>
            </div>
          </div>

          <div className="flex items-center gap-4 mt-3 text-xs text-gray-400">
            <span className="flex items-center gap-1"><Clock className="w-3.5 h-3.5" /> {restaurant.deliveryTimeMin}-{restaurant.deliveryTimeMax} دقيقة</span>
            <span className="flex items-center gap-1"><Truck className="w-3.5 h-3.5" /> {restaurant.deliveryFee === 0 ? 'مجاني' : `${(restaurant.deliveryFee ?? 0).toFixed(2)} ج.م`}</span>
          </div>
        </div>
      </div>

      {/* Search within menu */}
      <div className="container mx-auto px-4 mt-4">
        <div className="relative max-w-md">
          <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="ابحث في القائمة..." className="pr-9 h-10 rounded-lg bg-gray-50 border-gray-200 focus:bg-white" />
        </div>
      </div>

      {/* Menu */}
      <div className="container mx-auto px-4 py-6 max-w-2xl">
        {popular.length > 0 && (
          <div className="mb-8">
            <h3 className="text-base font-bold text-black mb-3">الأكثر طلباً</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {popular.map(item => <MenuCard key={item.id} item={item} onClick={() => openDetail(item)} />)}
            </div>
          </div>
        )}

        {rest.length > 0 && (
          <div>
            <h3 className="text-base font-bold text-black mb-3">القائمة</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {rest.map(item => <MenuCard key={item.id} item={item} onClick={() => openDetail(item)} />)}
            </div>
          </div>
        )}

        {filteredMenu.length === 0 && (
          <div className="text-center py-16">
            <p className="text-sm text-gray-400">لا توجد أصناف</p>
          </div>
        )}
      </div>

      <ProductDetail item={selectedItem} open={detailOpen} onOpenChange={setDetailOpen} />
    </div>
  )
}
