package qibla

import "math"

const (
	kaabaLat = 21.4225
	kaabaLon = 39.8262
	earthR   = 6371.0 //радиус земли, км
)

func rad(d float64) float64 {
	return d * math.Pi / 180
}

func deg(r float64) float64 {
	return r * 180 / math.Pi
}

// Direction возвращает азимут на Каабу в градусах (0–360) от истинного севера.
func Direction(lat, lon float64) float64 {
	φ1, φ2 := rad(lat), rad(kaabaLat)
	Δλ := rad(kaabaLon - lon)

	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)

	return math.Mod(deg(math.Atan2(y, x))+360, 360)
}

// Distance возвращает расстояние до Каабы в км (формула гаверсинусов).
func Distance(lat, lon float64) float64 {
	φ1, φ2 := rad(lat), rad(kaabaLat)
	Δφ := rad(kaabaLat - lat)
	Δλ := rad(kaabaLon - lon)

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return 2 * earthR * math.Asin(math.Sqrt(a))
}
