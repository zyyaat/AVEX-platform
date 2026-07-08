// Package domain category: Category entity for menu grouping.
package domain

import (
	"strings"
	"time"
)

type Category struct {
	id        string
	name      string
	nameAr    string
	icon      string
	imageURL  string
	sortOrder int
	createdAt time.Time
}

type CategoryParams struct {
	ID, Name, NameAr, Icon, ImageURL string
	SortOrder                        int
	Now                              time.Time
}

func NewCategory(p CategoryParams) (Category, error) {
	if p.ID == "" {
		return Category{}, NewValidationError("id", ErrInvalidID)
	}
	if strings.TrimSpace(p.Name) == "" {
		return Category{}, NewValidationError("name", ErrNameRequired)
	}
	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Category{id: p.ID, name: strings.TrimSpace(p.Name), nameAr: p.NameAr, icon: p.Icon, imageURL: p.ImageURL, sortOrder: p.SortOrder, createdAt: now}, nil
}

type CategoryRecord struct {
	ID, Name, NameAr, Icon, ImageURL string
	SortOrder                        int
	CreatedAt                        time.Time
}

func ReconstructCategory(r CategoryRecord) Category {
	return Category{id: r.ID, name: r.Name, nameAr: r.NameAr, icon: r.Icon, imageURL: r.ImageURL, sortOrder: r.SortOrder, createdAt: r.CreatedAt}
}

func (c Category) ID() string           { return c.id }
func (c Category) Name() string         { return c.name }
func (c Category) NameAr() string       { return c.nameAr }
func (c Category) Icon() string         { return c.icon }
func (c Category) ImageURL() string     { return c.imageURL }
func (c Category) SortOrder() int       { return c.sortOrder }
func (c Category) CreatedAt() time.Time { return c.createdAt }
