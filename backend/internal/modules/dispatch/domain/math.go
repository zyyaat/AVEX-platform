// Package domain math wrappers: thin shims around the math package.
//
// We wrap the math functions so the domain layer can use them without
// importing "math" directly in every file (cleaner imports).
package domain

import "math"

func mathSin(x float64) float64   { return math.Sin(x) }
func mathCos(x float64) float64   { return math.Cos(x) }
func mathAtan2(y, x float64) float64 { return math.Atan2(y, x) }
func mathSqrt(x float64) float64  { return math.Sqrt(x) }

// EarthRadiusMeters is the mean Earth radius in meters (WGS-84).
const EarthRadiusMeters = 6371000.0
