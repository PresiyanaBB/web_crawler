package crawler_app

import "web_image_crawler/model"

type ImageRepository interface {
	GetAll() ([]model.Image, error)
	Insert(img *model.Image) error
	DeleteAll() error
	FindBySiteName(site_name string) ([]model.Image, error)
	FindByResolution(resolution string) ([]model.Image, error)
	FindBySiteNameAndResolution(site_name string, resolution string) ([]model.Image, error)
}

type ImgCrawlerApp struct {
	images ImageRepository
}

func New(image ImageRepository) *ImgCrawlerApp {
	return &ImgCrawlerApp{
		images: image,
	}
}

func (cr *ImgCrawlerApp) GetAll() ([]model.Image, error) {
	return cr.images.GetAll()
}

func (cr *ImgCrawlerApp) Add(img *model.Image) error {
	return cr.images.Insert(img)
}

func (cr *ImgCrawlerApp) DeleteAll() error {
	return cr.images.DeleteAll()
}

func (cr *ImgCrawlerApp) FindBySiteName(site_name string) ([]model.Image, error) {
	return cr.images.FindBySiteName(site_name)
}

func (cr *ImgCrawlerApp) FindByResolution(resolution string) ([]model.Image, error) {
	return cr.images.FindByResolution(resolution)
}

func (cr *ImgCrawlerApp) FindBySiteNameAndResolution(site_name string, resolution string) ([]model.Image, error) {
	return cr.images.FindBySiteNameAndResolution(site_name, resolution)
}
