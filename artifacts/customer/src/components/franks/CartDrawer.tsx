
import { motion, AnimatePresence } from 'framer-motion'
import { X, Plus, Minus, Trash2, ShoppingBag, ArrowLeft } from 'lucide-react'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from '@/components/ui/sheet'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'
import { useCart } from '@/store/cart'
import { useState, useEffect } from 'react'
import { toast } from 'sonner'

interface CartDrawerProps { onCheckout: () => void }

export function CartDrawer({ onCheckout }: CartDrawerProps) {
  const { items, isOpen, setOpen, updateQuantity, removeItem, addItem, getSubtotal, getDeliveryFee, getTotal, getTotalItems } = useCart()
  const [recommended, setRecommended] = useState<any[]>([])
  const subtotal = getSubtotal()
  const deliveryFee = getDeliveryFee()
  const total = getTotal()
  const remaining = Math.max(0, 30 - subtotal)

  // Fetch recommended items
  useEffect(() => {
    if (isOpen && items.length > 0) {
      fetch('/api/v1/categories')
        .then(r => r.json())
        .then(data => {
          const allItems = (data.categories || []).flatMap((c: any) => c.items)
          // Items not in cart, preferably popular
          const cartIds = items.map(i => i.id.split('-')[0])
          const recs = allItems
            .filter((i: any) => !cartIds.includes(i.id))
            .sort((a: any, b: any) => (b.isPopular ? 1 : 0) - (a.isPopular ? 1 : 0))
            .slice(0, 4)
          setRecommended(recs)
        })
        .catch(() => {})
    }
  }, [isOpen, items])

  const handleAddRec = (item: any) => {
    addItem({ id: `${item.id}-rec-${Date.now()}`, name: item.name, nameAr: item.nameAr, price: item.price, image: item.imageUrl || item.image })
    toast.success(`تمت إضافة ${item.nameAr}`, { duration: 1500 })
  }

  return (
    <Sheet open={isOpen} onOpenChange={setOpen}>
      <SheetContent side="left" className="w-full sm:max-w-md flex flex-col p-0 gap-0" style={{ height: '100dvh', maxHeight: '100dvh' }}>
        <SheetHeader className="px-5 py-3.5 border-b border-gray-100 flex-shrink-0">
          <SheetTitle className="flex items-center gap-2 text-base font-bold">
            <ShoppingBag className="w-4 h-4" />
            سلة التسوق
            {getTotalItems() > 0 && <span className="text-xs text-gray-400">({getTotalItems()})</span>}
          </SheetTitle>
        </SheetHeader>

        {items.length === 0 ? (
          <div className="flex-1 flex flex-col items-center justify-center p-8 text-center gap-3">
            <div className="w-16 h-16 rounded-full bg-gray-50 flex items-center justify-center">
              <ShoppingBag className="w-8 h-8 text-gray-300" />
            </div>
            <div>
              <p className="font-medium text-black">سلتك فارغة</p>
              <p className="text-sm text-gray-400 mt-0.5">أضف بعض الأطباق لتبدأ</p>
            </div>
            <Button onClick={() => setOpen(false)} variant="ghost" size="sm" className="text-sm mt-1">تصفح القائمة</Button>
          </div>
        ) : (
          <>
            {/* Free delivery progress */}
            {remaining > 0 && (
              <div className="px-5 py-2 bg-gray-50 border-b border-gray-100">
                <p className="text-xs text-gray-500 text-center">أضف {remaining.toFixed(2)} ج.م للتوصيل المجاني</p>
              </div>
            )}

            <ScrollArea className="flex-1 min-h-0">
              <div className="p-4 space-y-3">
                {/* Cart items */}
                <AnimatePresence>
                  {items.map(item => (
                    <motion.div key={item.id} layout initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0, height: 0 }} className="flex gap-3 border border-gray-100 rounded-lg p-3">
                      <div className="w-14 h-14 rounded-md bg-gray-50 flex items-center justify-center overflow-hidden flex-shrink-0">
                        {item.image.startsWith('http') || item.image.startsWith('/') ? <img src={item.image} alt={item.nameAr} className="w-full h-full object-cover" /> : <span className="text-2xl">{item.image}</span>}
                      </div>
                      <div className="flex-1 min-w-0">
                        <h4 className="font-medium text-sm line-clamp-1">{item.nameAr}</h4>
                        <p className="text-xs text-gray-400">{item.price.toFixed(2)} ج.م</p>
                        <div className="flex items-center justify-between mt-2">
                          <div className="flex items-center gap-1 border border-gray-200 rounded-md">
                            <button onClick={() => updateQuantity(item.id, item.quantity - 1)} disabled={item.quantity <= 1} className={`w-6 h-6 flex items-center justify-center rounded ${item.quantity <= 1 ? 'text-gray-300 cursor-not-allowed' : 'hover:bg-gray-50'}`}><Minus className="w-3 h-3" /></button>
                            <span className="w-6 text-center text-sm font-medium">{item.quantity}</span>
                            <button onClick={() => updateQuantity(item.id, item.quantity + 1)} className="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-50"><Plus className="w-3 h-3" /></button>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-sm">{(item.price * item.quantity).toFixed(2)} ج.م</span>
                            <button onClick={() => removeItem(item.id)} className="text-gray-300 hover:text-black"><Trash2 className="w-3.5 h-3.5" /></button>
                          </div>
                        </div>
                      </div>
                    </motion.div>
                  ))}
                </AnimatePresence>

                {/* Recommended items */}
                {recommended.length > 0 && (
                  <div className="pt-2">
                    <h3 className="text-sm font-bold text-black mb-2">منتجات قد تهمك</h3>
                    <div className="grid grid-cols-2 gap-2">
                      {recommended.map(rec => (
                        <div key={rec.id} className="flex items-center gap-2 border border-gray-100 rounded-lg p-2">
                          <div className="w-10 h-10 rounded-md bg-gray-50 overflow-hidden flex-shrink-0">
                            {rec.imageUrl ? <img src={rec.imageUrl} alt={rec.nameAr} className="w-full h-full object-cover" /> : <span className="text-lg flex items-center justify-center w-full h-full">{rec.image}</span>}
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-black truncate">{rec.nameAr}</p>
                            <p className="text-xs text-gray-400">{rec.price.toFixed(2)} ج.م</p>
                          </div>
                          <button onClick={() => handleAddRec(rec)} className="w-7 h-7 rounded-md bg-gray-50 hover:bg-gray-100 flex items-center justify-center"><Plus className="w-3.5 h-3.5" /></button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </ScrollArea>

            <SheetFooter className="border-t border-gray-100 p-4 space-y-2 flex-shrink-0" style={{ paddingBottom: 'calc(1rem + env(safe-area-inset-bottom, 0px) + 40px)' }}>
              <div className="space-y-1 text-sm">
                <div className="flex justify-between text-gray-500"><span>المجموع الفرعي</span><span className="text-black">{subtotal.toFixed(2)} ج.م</span></div>
                <div className="flex justify-between text-gray-500"><span>التوصيل</span><span className="text-black">{deliveryFee === 0 ? <span className="text-gray-500">مجاني</span> : `${deliveryFee.toFixed(2)} ج.م`}</span></div>
                <Separator />
                <div className="flex justify-between font-bold pt-1"><span>الإجمالي</span><span>{total.toFixed(2)} ج.م</span></div>
              </div>
              <Button onClick={onCheckout} className="w-full h-11 rounded-lg bg-black hover:bg-gray-800 text-sm font-medium">متابعة لتأكيد الطلب <ArrowLeft className="w-4 h-4 mr-1" /></Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
