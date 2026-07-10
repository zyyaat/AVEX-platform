import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  UtensilsCrossed, Loader2, Plus, Power, Pencil, X, Save, Star, Trash2, Search,
} from 'lucide-react'
import { merchantAPI, type MenuItem, type Category } from '@/lib/api'
import { toast } from 'sonner'

const emptyForm: Partial<MenuItem> = {
  name: '',
  nameAr: '',
  description: '',
  descriptionAr: '',
  price: 0,
  image: '🍽️',
  imageUrl: '',
  categoryId: '',
  prepTime: 15,
  calories: 0,
  isPopular: false,
  isAvailable: true,
}

export default function MerchantMenuPage() {
  const [items, setItems] = useState<MenuItem[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState<Partial<MenuItem>>(emptyForm)
  const [search, setSearch] = useState('')
  const [saving, setSaving] = useState(false)

  const load = () => {
    setLoading(true)
    merchantAPI.getMenu().then((r) => {
      setItems(r.items || [])
      setCategories(r.categories || [])
    }).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.nameAr || !form.price) {
      toast.error('الاسم والسعر مطلوبان')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await merchantAPI.updateMenuItem(editing, form)
        toast.success('تم تحديث الصنف')
      } else {
        await merchantAPI.createMenuItem(form)
        toast.success('تمت إضافة الصنف')
      }
      setEditing(null)
      setShowCreate(false)
      setForm(emptyForm)
      load()
    } catch (err: any) {
      toast.error(err.message)
    } finally {
      setSaving(false)
    }
  }

  const toggleAvailable = async (it: MenuItem) => {
    try {
      await merchantAPI.updateMenuItem(it.id, { isAvailable: !it.isAvailable } as any)
      load()
      toast.success(it.isAvailable ? 'تم إخفاء الصنف' : 'تم إتاحة الصنف')
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const del = async (it: MenuItem) => {
    if (!confirm(`حذف "${it.nameAr}"؟`)) return
    try {
      await merchantAPI.deleteMenuItem(it.id)
      toast.success('تم الحذف')
      load()
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const openEdit = (it: MenuItem) => {
    setEditing(it.id)
    setForm({
      name: it.name,
      nameAr: it.nameAr,
      description: it.description,
      descriptionAr: it.descriptionAr,
      price: it.price,
      image: it.image,
      imageUrl: it.imageUrl,
      categoryId: it.categoryId,
      prepTime: it.prepTime,
      calories: it.calories,
      isPopular: it.isPopular,
      isAvailable: it.isAvailable,
    })
    setShowCreate(true)
  }

  const filteredItems = items.filter((it) =>
    !search || it.nameAr.includes(search) || it.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div dir="rtl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">المنيو ({items.length})</h1>
        <button
          onClick={() => { setEditing(null); setForm(emptyForm); setShowCreate(true) }}
          className="px-4 h-9 rounded-lg bg-black text-white text-sm font-bold flex items-center gap-2 hover:bg-gray-800 transition-fluent"
        >
          <Plus className="w-4 h-4" />
          صنف جديد
        </button>
      </div>

      {/* Search */}
      <div className="relative mb-4">
        <Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="ابحث في المنيو..."
          className="w-full h-11 pr-10 pl-4 rounded-xl border border-gray-200 bg-white focus:outline-none focus:border-black focus:ring-1 focus:ring-black transition-fluent"
        />
      </div>

      {/* Items */}
      {loading ? (
        <div className="py-20 text-center"><Loader2 className="w-6 h-6 animate-spin mx-auto" /></div>
      ) : filteredItems.length === 0 ? (
        <div className="text-center py-20">
          <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mx-auto mb-4">
            <UtensilsCrossed className="w-7 h-7 text-gray-300" />
          </div>
          <p className="text-sm text-gray-500 font-medium">{items.length === 0 ? 'لا توجد أصناف بعد' : 'لا نتائج'}</p>
          {items.length === 0 && (
            <p className="text-xs text-gray-400 mt-1">اضغط "صنف جديد" لإضافة أول صنف</p>
          )}
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-3">
          {filteredItems.map((it, idx) => (
            <motion.div
              key={it.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: idx * 0.03 }}
              className={`bg-white rounded-xl border border-gray-200 p-3 flex gap-3 shadow-fluent ${!it.isAvailable ? 'opacity-60' : ''}`}
            >
              {/* Image */}
              <div className="w-16 h-16 rounded-lg bg-gray-100 flex-shrink-0 overflow-hidden">
                {it.imageUrl ? (
                  <img src={it.imageUrl} alt={it.nameAr} className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-2xl">{it.image || '🍽️'}</div>
                )}
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="font-bold text-sm flex items-center gap-1 truncate">
                      {it.nameAr}
                      {it.isPopular && <Star className="w-3 h-3 fill-black text-black flex-shrink-0" />}
                    </p>
                    <p className="text-[10px] text-gray-500 line-clamp-1">{it.descriptionAr}</p>
                  </div>
                  <span className={`text-[10px] px-2 py-0.5 rounded-full flex-shrink-0 ${
                    it.isAvailable ? 'bg-black text-white' : 'bg-gray-200 text-gray-500'
                  }`}>
                    {it.isAvailable ? 'متاح' : 'مخفي'}
                  </span>
                </div>
                <div className="flex items-center justify-between mt-1">
                  <p className="font-bold text-sm">{(it.price ?? 0).toFixed(2)} ج.م</p>
                  <p className="text-[10px] text-gray-400">{it.prepTime} دقيقة</p>
                </div>
                <div className="flex gap-1 mt-2">
                  <button
                    onClick={() => openEdit(it)}
                    className="flex-1 h-7 rounded border border-gray-200 text-[10px] font-bold flex items-center justify-center gap-1 hover:bg-gray-50 transition-fluent"
                  >
                    <Pencil className="w-2.5 h-2.5" />
                    تعديل
                  </button>
                  <button
                    onClick={() => toggleAvailable(it)}
                    className="flex-1 h-7 rounded border border-gray-200 text-[10px] font-bold flex items-center justify-center gap-1 hover:bg-gray-50 transition-fluent"
                  >
                    <Power className="w-2.5 h-2.5" />
                    {it.isAvailable ? 'إخفاء' : 'إتاحة'}
                  </button>
                  <button
                    onClick={() => del(it)}
                    className="w-7 h-7 rounded border border-gray-200 text-gray-500 hover:bg-gray-50 transition-fluent flex items-center justify-center"
                    aria-label="حذف"
                  >
                    <Trash2 className="w-2.5 h-2.5" />
                  </button>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      )}

      {/* Create/Edit modal */}
      <AnimatePresence>
        {showCreate && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/60 flex items-end sm:items-center justify-center p-0 sm:p-4"
            onClick={(e) => e.target === e.currentTarget && (setShowCreate(false), setEditing(null))}
          >
            <motion.div
              initial={{ y: '100%', opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: '100%', opacity: 0 }}
              transition={{ type: 'spring', damping: 28, stiffness: 320 }}
              className="bg-white w-full sm:max-w-md sm:rounded-2xl rounded-t-2xl max-h-[90vh] overflow-y-auto"
              dir="rtl"
            >
              <div className="bg-black text-white px-5 py-4 flex items-center justify-between sticky top-0">
                <h3 className="font-bold">{editing ? 'تعديل صنف' : 'صنف جديد'}</h3>
                <button
                  onClick={() => { setShowCreate(false); setEditing(null) }}
                  className="w-8 h-8 rounded-full hover:bg-white/10 flex items-center justify-center"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={save} className="p-5 space-y-3">
                <FormField label="الاسم بالعربية *">
                  <input
                    required
                    value={form.nameAr || ''}
                    onChange={(e) => setForm({ ...form, nameAr: e.target.value })}
                    className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                  />
                </FormField>

                <FormField label="الاسم بالإنجليزية">
                  <input
                    value={form.name || ''}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                  />
                </FormField>

                <FormField label="الوصف بالعربية">
                  <textarea
                    value={form.descriptionAr || ''}
                    onChange={(e) => setForm({ ...form, descriptionAr: e.target.value })}
                    rows={2}
                    className="w-full p-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black resize-none"
                  />
                </FormField>

                <div className="grid grid-cols-2 gap-2">
                  <FormField label="السعر (ج.م) *">
                    <input
                      type="number"
                      step="0.01"
                      required
                      value={form.price || ''}
                      onChange={(e) => setForm({ ...form, price: +e.target.value })}
                      className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                    />
                  </FormField>
                  <FormField label="الفئة">
                    <select
                      value={form.categoryId || ''}
                      onChange={(e) => setForm({ ...form, categoryId: e.target.value })}
                      className="w-full h-11 px-3 rounded-lg border border-gray-200 bg-white focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                    >
                      <option value="">اختر الفئة</option>
                      {categories.map((c) => (
                        <option key={c.id} value={c.id}>{c.nameAr}</option>
                      ))}
                    </select>
                  </FormField>
                </div>

                <FormField label="رابط الصورة">
                  <input
                    value={form.imageUrl || ''}
                    onChange={(e) => setForm({ ...form, imageUrl: e.target.value })}
                    placeholder="https://..."
                    className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                  />
                </FormField>

                <div className="grid grid-cols-2 gap-2">
                  <FormField label="وقت التحضير (دقيقة)">
                    <input
                      type="number"
                      value={form.prepTime || ''}
                      onChange={(e) => setForm({ ...form, prepTime: +e.target.value })}
                      className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                    />
                  </FormField>
                  <FormField label="السعرات">
                    <input
                      type="number"
                      value={form.calories || ''}
                      onChange={(e) => setForm({ ...form, calories: +e.target.value })}
                      className="w-full h-11 px-3 rounded-lg border border-gray-200 focus:outline-none focus:border-black focus:ring-1 focus:ring-black"
                    />
                  </FormField>
                </div>

                <div className="flex gap-4 pt-2">
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form.isPopular || false}
                      onChange={(e) => setForm({ ...form, isPopular: e.target.checked })}
                      className="w-4 h-4"
                    />
                    شائع
                  </label>
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form.isAvailable !== false}
                      onChange={(e) => setForm({ ...form, isAvailable: e.target.checked })}
                      className="w-4 h-4"
                    />
                    متاح للطلب
                  </label>
                </div>

                <button
                  type="submit"
                  disabled={saving}
                  className="w-full h-12 rounded-xl bg-black text-white font-bold flex items-center justify-center gap-2 hover:bg-gray-800 transition-fluent disabled:opacity-50 mt-3"
                >
                  {saving ? <Loader2 className="w-5 h-5 animate-spin" /> : <Save className="w-4 h-4" />}
                  {editing ? 'حفظ' : 'إضافة'}
                </button>
              </form>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-xs text-gray-500 mb-1 block">{label}</label>
      {children}
    </div>
  )
}
