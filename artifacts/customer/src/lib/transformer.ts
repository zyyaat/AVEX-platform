// AVEX Shared — API response transformer
// =============================================================================
// The Go backend serializes all DTOs with snake_case JSON tags (e.g.
// `order_number`, `is_admin`, `created_at`). The frontend TypeScript types
// use camelCase (e.g. `orderNumber`, `isAdmin`, `createdAt`).
//
// This transformer recursively converts snake_case keys to camelCase in
// any JSON-compatible value. It is applied automatically by apiFetch()
// to every successful response, so frontend code can use camelCase
// field names directly.
//
// Usage:
//   import { toCamelCase } from './transformer'
//   const data = toCamelCase(json.data)
//
// Notes:
//   - Arrays are transformed element-by-element.
//   - null and primitives are returned as-is.
//   - Date strings are NOT parsed (keep them as strings; use new Date()
//     at the call site if needed).
//   - Keys that are already camelCase are left unchanged.
//   - Keys with leading underscores (e.g. "_internal") are preserved.
// =============================================================================

/**
 * Convert a snake_case string to camelCase.
 * Examples:
 *   "order_number" → "orderNumber"
 *   "is_admin"     → "isAdmin"
 *   "created_at"   → "createdAt"
 *   "user_id"      → "userId"
 *   "id"           → "id"  (no change)
 */
export function snakeToCamel(key: string): string {
  // Don't transform keys that don't contain an underscore,
  // or that start with an underscore (internal fields).
  if (!key.includes('_') || key.startsWith('_')) {
    return key
  }
  return key.replace(/_([a-z0-9])/g, (_, char) => char.toUpperCase())
}

/**
 * Recursively transform all keys in a JSON-compatible value from
 * snake_case to camelCase.
 *
 * Handles: objects, arrays, null, primitives.
 * Does NOT mutate the input — returns a new value.
 */
export function toCamelCase<T>(value: unknown): T {
  if (value === null || value === undefined) {
    return value as T
  }

  if (Array.isArray(value)) {
    return value.map((item) => toCamelCase(item)) as unknown as T
  }

  if (typeof value === 'object' && value instanceof Date) {
    return value as T
  }

  if (typeof value === 'object') {
    const result: Record<string, unknown> = {}
    for (const [key, val] of Object.entries(value as Record<string, unknown>)) {
      result[snakeToCamel(key)] = toCamelCase(val)
    }
    return result as T
  }

  // primitives: string, number, boolean
  return value as T
}
