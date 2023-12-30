package mysql

import (
	"database/sql"
	"fmt"
	"web_image_crawler/model"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLRepository struct {
	opts   MySQLOptions
	client *sql.DB
}

type MySQLOptions struct {
	URI string
}

func New(opts MySQLOptions) *MySQLRepository {
	return &MySQLRepository{
		opts:   opts,
		client: nil,
	}
}

func (r *MySQLRepository) Init() error {
	var err error
	r.client, err = sql.Open("mysql", r.opts.URI)
	return err
}

func (r *MySQLRepository) GetAll() ([]model.Image, error) {
	if r.client == nil {
		return nil, fmt.Errorf("mysql repository is not initilized")
	}
	var images []model.Image

	rows, err := r.client.Query("SELECT * FROM images")
	if err != nil {
		return nil, fmt.Errorf("mysql query failure: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result model.Image
		rows.Scan(&result.ID, &result.Filename, &result.AlternativeText, &result.Src, &result.Resolution, &result.Format)
		images = append(images, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating images: %w", err)
	}
	return images, nil
}

func (r *MySQLRepository) Insert(img *model.Image) error {
	if r.client == nil {
		return fmt.Errorf("mysql repository is not initilized")
	}

	_, err := r.client.Exec("INSERT INTO images(id, filename, alternative_text, src, resolution, format) VALUES (?, ?,?, ?,?, ?)",
		img.ID, img.Filename, img.AlternativeText, img.Src, img.Resolution, img.Format)

	return err
}

func (r *MySQLRepository) DeleteAll() error {
	if r.client == nil {
		return fmt.Errorf("mysql repository is not initilized")
	}
	_, err := r.client.Exec("TRUNCATE TABLE images")

	return err
}

func (r *MySQLRepository) FindBySiteName(site_name string) ([]model.Image, error) {
	query := fmt.Sprintf("select * from images where filename like '%v%v%v'", "%", site_name, "%")
	if r.client == nil {
		return nil, fmt.Errorf("mysql repository is not initilized")
	}
	var images []model.Image

	rows, err := r.client.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql query failure: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result model.Image
		rows.Scan(&result.ID, &result.Filename, &result.AlternativeText, &result.Src, &result.Resolution, &result.Format)
		images = append(images, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating images: %w", err)
	}
	return images, nil
}

func (r *MySQLRepository) FindByResolution(resolution string) ([]model.Image, error) {
	query := fmt.Sprintf("select * from images where resolution like '%v%v%v'", "%", resolution, "%")
	if r.client == nil {
		return nil, fmt.Errorf("mysql repository is not initilized")
	}
	var images []model.Image

	rows, err := r.client.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql query failure: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result model.Image
		rows.Scan(&result.ID, &result.Filename, &result.AlternativeText, &result.Src, &result.Resolution, &result.Format)
		images = append(images, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating images: %w", err)
	}
	return images, nil
}

func (r *MySQLRepository) FindBySiteNameAndResolution(site_name string, resolution string) ([]model.Image, error) {
	query := fmt.Sprintf("select * from images where filename like '%v%v%v' and resolution like '%v%v%v'", "%", site_name, "%", "%", resolution, "%")
	if r.client == nil {
		return nil, fmt.Errorf("mysql repository is not initilized")
	}
	var images []model.Image

	rows, err := r.client.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql query failure: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result model.Image
		rows.Scan(&result.ID, &result.Filename, &result.AlternativeText, &result.Src, &result.Resolution, &result.Format)
		images = append(images, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating images: %w", err)
	}
	return images, nil
}
