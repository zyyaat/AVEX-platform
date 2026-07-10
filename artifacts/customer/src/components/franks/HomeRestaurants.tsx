
import { useState, useEffect } from 'react'
import { Search, Star, Clock, ChevronLeft, Loader2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { useRouter } from '@/lib/navigation'

interface Restaurant {
  id: string
  name: string
  nameAr: string
  descriptionAr: string
  imageUrl: string
  rating: number
  ratingCount: number
  deliveryTimeMin: number
  deliveryTimeMax: number
  deliveryFee: number
  minOrder: number
  isPro: boolean
  cuisines: string
}

export function HomeRestaurants() {
  const [restaurants, setRestaurants] = useState<Restaurant[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const router = useRouter()

  useEffect(() => {
    fetch('/api/v1/restaurants')
      .then(r => r.json())
      .then(data => { setRestaurants(data.restaurants || []); setLoading(false) })
      .catch(() => setLoading(false))
  }, [])

  const filtered = restaurants.filter(r =>
    !search ||
    r.nameAr.includes(search) ||
    r.name.toLowerCase().includes(search.toLowerCase()) ||
    r.cuisines.includes(search)
  )

  const proRestaurants = filtered.filter(r => r.isPro)
  const regularRestaurants = filtered.filter(r => !r.isPro)

  return (
    <section className="py-6 md:py-8 bg-white">
      <div className="container mx-auto px-4">
        {/* Search */}
        <div className="max-w-md mx-auto mb-6 relative">
          <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="ابحث عن مطعم أو نوع طعام..."
            className="pr-9 h-10 rounded-lg bg-gray-50 border-gray-200 focus:bg-white"
          />
        </div>

        {loading ? (
          <div className="space-y-3">
            {[1,2,3].map(i => <div key={i} className="h-28 rounded-lg bg-gray-50 animate-pulse" />)}
          </div>
        ) : (
          <div className="space-y-8 max-w-2xl mx-auto">
            {/* Pro restaurants */}
            {proRestaurants.length > 0 && !search && (
              <div>
                <h3 className="text-base font-bold text-black mb-3">مطاعم مميزة</h3>
                <div className="space-y-3">
                  {proRestaurants.map(r => <RestaurantCard key={r.id} restaurant={r} onClick={() => router.push(`/?restaurant=${r.id}`)} />)}
                </div>
              </div>
            )}

            {/* All restaurants */}
            <div>
              <h3 className="text-base font-bold text-black mb-3">
                {search ? `${filtered.length} نتيجة` : 'كل المطاعم'}
              </h3>
              <div className="space-y-3">
                {(search ? filtered : regularRestaurants).map(r => (
                  <RestaurantCard key={r.id} restaurant={r} onClick={() => router.push(`/?restaurant=${r.id}`)} />
                ))}
              </div>
            </div>

            {filtered.length === 0 && (
              <div className="text-center py-16">
                <div className="w-14 h-14 rounded-full bg-gray-50 flex items-center justify-center mx-auto mb-3">
                  <Search className="w-6 h-6 text-gray-300" />
                </div>
                <p className="text-sm font-medium text-black">لا توجد مطاعم</p>
                <p className="text-xs text-gray-400 mt-1">جرّب كلمات بحث أخرى</p>
              </div>
            )}
          </div>
        )}
      </div>
    </section>
  )
}

function RestaurantCard({ restaurant, onClick }: { restaurant: Restaurant; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      className="group flex gap-3 bg-white rounded-lg border border-gray-200 p-3 hover:border-gray-300 hover:shadow-fluent transition-fluent cursor-pointer"
    >
      {/* Image */}
      <div className="w-20 h-20 rounded-md bg-gray-50 overflow-hidden flex-shrink-0">
        {restaurant.imageUrl ? (
          <img src={restaurant.imageUrl} alt={restaurant.nameAr} className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-2xl">🍽️</div>
        )}
      </div>

      {/* Info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <h3 className="font-medium text-sm text-black truncate">{restaurant.nameAr}</h3>
              {restaurant.isPro && (
                <span className="text-[9px] bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded font-medium">PRO</span>
              )}
            </div>
            <p className="text-xs text-gray-400 truncate mt-0.5">{restaurant.cuisines}</p>
          </div>
          <div className="flex items-center gap-1 bg-gray-50 rounded px-1.5 py-0.5 flex-shrink-0">
            <Star className="w-3 h-3 fill-black text-black" />
            <span className="text-xs font-medium text-black">{restaurant.rating}</span>
            <span className="text-[10px] text-gray-400">({restaurant.ratingCount})</span>
          </div>
        </div>

        <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
          <span className="flex items-center gap-1">
            <Clock className="w-3 h-3" />
            {restaurant.deliveryTimeMin}-{restaurant.deliveryTimeMax} د
          </span>
          <span>
            {restaurant.deliveryFee === 0 ? (
              <span className="text-gray-500">توصيل مجاني</span>
            ) : (
              `${(restaurant.deliveryFee ?? 0).toFixed(2)} ج.م توصيل`
            )}
          </span>
        </div>

        {restaurant.descriptionAr && (
          <p className="text-xs text-gray-400 mt-1 line-clamp-1">{restaurant.descriptionAr}</p>
        )}
      </div>

      <ChevronLeft className="w-4 h-4 text-gray-300 group-hover:text-gray-400 flex-shrink-0 self-center" />
    </div>
  )
}
