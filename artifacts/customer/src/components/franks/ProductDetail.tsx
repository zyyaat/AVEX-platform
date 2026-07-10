
import { useState, useEffect } from 'react'
import { X, Star, Plus, Minus, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Drawer, DrawerContent } from '@/components/ui/drawer'
import { useCart } from '@/store/cart'
import { toast } from 'sonner'

interface MenuItemType {
  id: string; name: string; nameAr: string; description: string; descriptionAr: string
  price: number; image: string; imageUrl: string | null; isPopular: boolean
  rating: number; ratingCount?: number; prepTime: number; calories: number
}

interface ProductDetailProps {
  item: MenuItemType | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

const SIZES = [
  { id: 'small', label: 'صغير', multiplier: 1 },
  { id: 'medium', label: 'متوسط', multiplier: 1.3 },
  { id: 'large', label: 'كبير', multiplier: 1.6 },
]

// Add-on suggestions (fetched from API)
interface AddOn { id: string; name: string; price: number; image: string; imageUrl: string | null }

export function ProductDetail({ item, open, onOpenChange }: ProductDetailProps) {
  const { addItem } = useCart()
  const [quantity, setQuantity] = useState(1)
  const [selectedSize, setSelectedSize] = useState('small')
  const [selectedAddOns, setSelectedAddOns] = useState<string[]>([])
  const [addOns, setAddOns] = useState<AddOn[]>([])

  // Reset state + fetch add-ons when item changes
  useEffect(() => {
    if (!item) return
    const timer = setTimeout(() => {
      setQuantity(1)
      setSelectedSize('small')
      setSelectedAddOns([])
    }, 0)

    fetch('/api/v1/categories')
      .then(r => r.json())
      .then(data => {
        const allItems = (data.categories || []).flatMap((c: any) => (c.items || []))
        const suggestions = allItems
          .filter((i: any) => i.id !== item.id && (i.calories === 0 || i.price < 8))
          .slice(0, 4)
          .map((i: any) => ({ id: i.id, name: i.nameAr, price: i.price, image: i.image, imageUrl: i.imageUrl }))
        setAddOns(suggestions)
      })
      .catch(() => {})

    return () => clearTimeout(timer)
  }, [item?.id])

  if (!item) return null

  const sizeConfig = SIZES.find(s => s.id === selectedSize) || SIZES[0]
  const basePrice = item.price * sizeConfig.multiplier
  const addOnsTotal = selectedAddOns.reduce((sum, id) => {
    const ao = addOns.find(a => a.id === id)
    return sum + (ao ? ao.price : 0)
  }, 0)
  const unitPrice = basePrice + addOnsTotal
  const totalPrice = unitPrice * quantity

  const toggleAddOn = (id: string) => {
    setSelectedAddOns(prev => prev.includes(id) ? prev.filter(a => a !== id) : [...prev, id])
  }

  const handleAdd = () => {
    const sizeLabel = SIZES.find(s => s.id === selectedSize)?.label || ''
    const customName = selectedSize !== 'small' ? `${item.nameAr} (${sizeLabel})` : item.nameAr

    for (let i = 0; i < quantity; i++) {
      addItem({ id: `${item.id}-${selectedSize}-${Date.now()}-${i}`, name: item.name, nameAr: customName, price: unitPrice, image: item.imageUrl || item.image })
    }
    toast.success(`تمت إضافة ${quantity}× ${customName}`, { duration: 1500 })
    onOpenChange(false)
  }

  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent className="max-h-[88dvh] rounded-t-2xl">
        <div className="mx-auto w-full max-w-md overflow-y-auto">
          {/* Image */}
          <div className="relative aspect-[4/3] w-full bg-gray-50">
            {item.imageUrl ? (
              <img src={item.imageUrl} alt={item.nameAr} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <span className="text-6xl text-gray-300">{item.image}</span>
              </div>
            )}
            <button onClick={() => onOpenChange(false)} className="absolute top-3 right-3 w-8 h-8 rounded-full bg-white/95 backdrop-blur-sm flex items-center justify-center shadow-fluent">
              <X className="w-4 h-4 text-black" />
            </button>
            {item.isPopular && (
              <div className="absolute top-3 left-3 bg-black text-white text-xs font-medium px-2 py-1 rounded">الأكثر طلباً</div>
            )}
          </div>

          <div className="p-5 space-y-5">
            {/* Title + rating */}
            <div>
              <div className="flex items-start justify-between gap-2">
                <h2 className="text-lg font-bold text-black">{item.nameAr}</h2>
                <div className="flex items-center gap-1 bg-gray-50 rounded-md px-2 py-1">
                  <Star className="w-3.5 h-3.5 fill-black text-black" />
                  <span className="text-sm font-medium">{item.rating}</span>
                  {item.ratingCount && <span className="text-xs text-gray-400">({item.ratingCount})</span>}
                </div>
              </div>
              <p className="text-sm text-gray-500 mt-1 leading-relaxed">{item.descriptionAr}</p>
            </div>

            {/* Sizes */}
            <div>
              <h3 className="text-sm font-bold text-black mb-2">ال حجم</h3>
              <div className="grid grid-cols-3 gap-2">
                {SIZES.map(size => {
                  const price = item.price * size.multiplier
                  const isSelected = selectedSize === size.id
                  return (
                    <button
                      key={size.id}
                      onClick={() => setSelectedSize(size.id)}
                      className={`relative p-3 rounded-lg border text-center transition-fluent ${
                        isSelected ? 'border-black bg-gray-50' : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      {isSelected && <div className="absolute top-1.5 left-1.5 w-4 h-4 rounded-full bg-black flex items-center justify-center"><Check className="w-2.5 h-2.5 text-white" /></div>}
                      <p className="text-sm font-medium text-black">{size.label}</p>
                      <p className="text-xs text-gray-400 mt-0.5">{price.toFixed(2)} ج.م</p>
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Add-ons */}
            {addOns.length > 0 && (
              <div>
                <h3 className="text-sm font-bold text-black mb-1">أضف إضافات</h3>
                <p className="text-xs text-gray-400 mb-3">يمكنك إضافة إضافات على هذا الطبق</p>
                <div className="space-y-2">
                  {addOns.map(ao => {
                    const isSelected = selectedAddOns.includes(ao.id)
                    return (
                      <div key={ao.id} className="flex items-center gap-3 p-2 rounded-lg border border-gray-100 hover:border-gray-200">
                        <div className="w-10 h-10 rounded-md bg-gray-50 overflow-hidden flex-shrink-0">
                          {ao.imageUrl ? <img src={ao.imageUrl} alt={ao.name} className="w-full h-full object-cover" /> : <span className="text-lg flex items-center justify-center w-full h-full">{ao.image}</span>}
                        </div>
                        <span className="flex-1 text-sm text-black">{ao.name}</span>
                        <span className="text-xs text-gray-400">+{ao.price.toFixed(2)} ج.م</span>
                        <button onClick={() => toggleAddOn(ao.id)} className={`w-7 h-7 rounded-md flex items-center justify-center transition-fluent ${isSelected ? 'bg-black' : 'border border-gray-200 hover:bg-gray-50'}`}>
                          {isSelected ? <Check className="w-3.5 h-3.5 text-white" /> : <Plus className="w-3.5 h-3.5 text-black" />}
                        </button>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {/* Quantity + Add button */}
            <div className="flex items-center justify-between pt-2 border-t border-gray-100">
              <div className="flex items-center gap-3">
                <button onClick={() => setQuantity(q => Math.max(1, q - 1))} disabled={quantity <= 1} className="w-9 h-9 rounded-lg border border-gray-200 flex items-center justify-center disabled:opacity-30 hover:bg-gray-50"><Minus className="w-4 h-4" /></button>
                <span className="w-8 text-center font-bold text-base">{quantity}</span>
                <button onClick={() => setQuantity(q => q + 1)} className="w-9 h-9 rounded-lg border border-gray-200 flex items-center justify-center hover:bg-gray-50"><Plus className="w-4 h-4" /></button>
              </div>
              <Button onClick={handleAdd} className="bg-black hover:bg-gray-800 text-white rounded-lg h-11 px-5 text-sm font-medium flex items-center gap-2">
                إضافة للسلة • {totalPrice.toFixed(2)} ج.م
              </Button>
            </div>
          </div>
        </div>
      </DrawerContent>
    </Drawer>
  )
}
