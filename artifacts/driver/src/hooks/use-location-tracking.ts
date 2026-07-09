import { useEffect, useRef } from 'react'
import { useDriver } from '@/store/driver'

interface UseLocationTrackingOptions {
  enabled: boolean
  interval?: number // ms (default 5000)
  highAccuracy?: boolean
}

/**
 * Tracks the driver's GPS location and sends updates to the backend.
 * 
 * Uses watchPosition for real-time updates, with a minimum interval
 * to avoid flooding the backend.
 * 
 * When enabled=false, tracking stops.
 */
export function useLocationTracking({
  enabled,
  interval = 5000,
  highAccuracy = true,
}: UseLocationTrackingOptions) {
  const { updateLocation } = useDriver()
  const watchIdRef = useRef<number | null>(null)
  const lastUpdateRef = useRef<number>(0)

  useEffect(() => {
    if (!enabled) {
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current)
        watchIdRef.current = null
      }
      return
    }

    if (!navigator.geolocation) {
      console.error('Geolocation not supported')
      return
    }

    watchIdRef.current = navigator.geolocation.watchPosition(
      (pos) => {
        const now = Date.now()
        // Throttle to the specified interval
        if (now - lastUpdateRef.current < interval) return
        lastUpdateRef.current = now

        updateLocation(
          pos.coords.latitude,
          pos.coords.longitude,
          pos.coords.heading || 0,
          pos.coords.speed || 0,
          pos.coords.accuracy || 0,
        )
      },
      (err) => {
        console.error('Geolocation error:', err.message)
      },
      {
        enableHighAccuracy: highAccuracy,
        timeout: 10000,
        maximumAge: interval,
      }
    )

    return () => {
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current)
        watchIdRef.current = null
      }
    }
  }, [enabled, interval, highAccuracy, updateLocation])
}
