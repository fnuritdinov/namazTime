package geo

import (
	"errors"
	"fmt"

	"github.com/sams96/rgeo"
	"github.com/twpayne/go-geom"
)

type Locator struct {
	r *rgeo.Rgeo
}

func New() (*Locator, error) {
	r, err := rgeo.New(rgeo.Countries10)
	if err != nil {
		return nil, fmt.Errorf("rgeo load: %w", err)
	}
	return &Locator{r: r}, nil
}

func (l *Locator) Country(lat, lon float64) string {
	loc, err := l.r.ReverseGeocode(geom.Coord{lon, lat})
	if errors.Is(err, rgeo.ErrLocationNotFound) || err != nil {
		return ""
	}
	return loc.CountryCode2
}
