package model

import "html/template"

type Image struct {
	ID              string
	Filename        string
	AlternativeText string
	Src             template.URL
	Resolution      string
	Format          string
}
