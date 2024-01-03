package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"web_image_crawler/crawler_app"
	"web_image_crawler/model"
	"web_image_crawler/mysql"

	"golang.org/x/net/html"

	"github.com/google/uuid"
)

var guard chan struct{}
var ch chan string
var stop_time time.Time
var MAX_GOROUTINES_COUNT int
var seconds int
var input_link string
var crawl_inner bool

type crawler struct {
	app           *crawler_app.ImgCrawlerApp
	server        *http.Server
	mux           *http.ServeMux
	templateIndex *template.Template
	templateShow  *template.Template
}

func FindAttribute(doc *html.Node, attr string) ([]*html.Node, error) {
	nodes := make([]*html.Node, 0)
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == attr {
			nodes = append(nodes, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)
	if nodes[0] != nil {
		return nodes, nil
	}
	return nil, errors.New("Missing")
}

func ExtractLinks(doc *html.Node) {
	if stop_time.Compare(time.Now()) <= 0 {
		return
	}
	linkNodes, err := FindAttribute(doc, "a")
	if err != nil {
		return
	}

	for _, links := range linkNodes {
		for _, v := range links.Attr {
			switch v.Key {
			case "href":
				if len(ch) == cap(ch) {
					return
				} else {
					ch <- v.Val
				}
			}
		}
	}
}

func ExtractImages(cr *crawler, doc *html.Node) {
	for len(guard) == MAX_GOROUTINES_COUNT {
	}
	guard <- struct{}{}
	if stop_time.Compare(time.Now()) <= 0 {
		<-guard
		return
	}
	node, err := FindAttribute(doc, "title")
	if err != nil {
		<-guard
		return
	}
	entries, err := FindAttribute(doc, "img")
	if err != nil {
		<-guard
		return
	}

	for _, entry := range entries {
		if stop_time.Compare(time.Now()) <= 0 {
			<-guard
			return
		}
		var src, alt, width, height string
		for _, v := range entry.Attr {
			switch v.Key {
			case "alt":
				alt = v.Val
			case "src":
				src = v.Val
			case "width":
				width = v.Val
			case "height":
				height = v.Val
			}
		}

		src_split := strings.Split(src, ".")

		img := model.Image{
			ID:              uuid.New().String(),
			Filename:        node[0].FirstChild.Data,
			AlternativeText: alt,
			Src:             template.URL(src),
			Resolution:      fmt.Sprintf("%vx%v", width, height),
			Format:          src_split[len(src_split)-1],
		}
		if stop_time.Compare(time.Now()) <= 0 {
			<-guard
			return
		}
		for len(guard) == MAX_GOROUTINES_COUNT {
		}
		guard <- struct{}{}
		go cr.app.Add(&img)
		<-guard
	}
	<-guard
}

func WebCrawler(cr *crawler, shouldCrawlInternalLinks bool, writer http.ResponseWriter, request *http.Request) {
	for {
		if len(ch) == 0 || stop_time.Compare(time.Now()) <= 0 {
			break
		}

		link := <-ch
		req, err := http.NewRequest("GET", link, nil)
		if err != nil {
			log.Printf("error opening link: %v", err)
			WebCrawler(cr, shouldCrawlInternalLinks, writer, request)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("error opening link: %v", err)
			WebCrawler(cr, shouldCrawlInternalLinks, writer, request)
			return
		}

		doc, _ := html.Parse(resp.Body)
		for len(guard) == cap(guard) {
			time.Sleep(1 * time.Millisecond)
		}
		if stop_time.Compare(time.Now()) <= 0 {
			http.Redirect(writer, request, "/show", http.StatusFound)
		}
		go ExtractImages(cr, doc)
		if stop_time.Compare(time.Now()) <= 0 {
			return
		}
		if shouldCrawlInternalLinks {
			for len(guard) == cap(guard) {
				if stop_time.Compare(time.Now()) <= 0 {
					http.Redirect(writer, request, "/show", http.StatusFound)
				}
				time.Sleep(1 * time.Millisecond)
			}
			if stop_time.Compare(time.Now()) <= 0 {
				http.Redirect(writer, request, "/show", http.StatusFound)
			}
			ExtractLinks(doc)
			if stop_time.Compare(time.Now()) <= 0 {
				return
			}
		}
		resp.Body.Close()
	}
}

func (cr *crawler) Run() error {
	cr.mux.HandleFunc("/", cr.handleMain)
	cr.mux.HandleFunc("/crawl", cr.handleCrawl)
	cr.mux.HandleFunc("/show", cr.handleShow)
	cr.templateIndex = template.Must(template.ParseFiles("./templ/index.html"))
	cr.templateShow = template.Must(template.ParseFiles("./templ/images.html"))
	log.Printf("server is listening at %s\n", cr.server.Addr)
	if err := cr.server.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("failed to start service on port %s:%w", cr.server.Addr, err))
		fmt.Print(cr.server)
		return nil
	}
	return nil
}

func (cr *crawler) handleMain(writer http.ResponseWriter, request *http.Request) {
	cr.templateIndex.Execute(writer, struct{}{})
}

func (cr *crawler) handleCrawl(writer http.ResponseWriter, request *http.Request) {
	defer http.Redirect(writer, request, "/show", http.StatusFound)
	if err := cr.app.DeleteAll(); err != nil {
		log.Printf("error parsing image: %v", err)
		return
	}

	time_lim := time.Duration(seconds) * time.Second
	stop_time = time.Now().Local().Add(time_lim)

	guard = make(chan struct{}, MAX_GOROUTINES_COUNT)
	ch = make(chan string, 1024)
	ch <- input_link
	WebCrawler(cr, crawl_inner, writer, request)
}

func SaveImagesToFileDirectory(images []model.Image) error {
	for index, i := range images {
		response, err := http.Get(fmt.Sprintf("https://%v", string(i.Src)))
		if err != nil {
			response, err = http.Get(fmt.Sprintf("https:/%v", string(i.Src)))
			if err != nil {
				response, err = http.Get(fmt.Sprintf("https:%v", string(i.Src)))
				if err != nil {
					continue
				}
			}
		}
		err = os.MkdirAll("image_found", os.ModePerm)
		if err != nil {
			return err
		}

		defer response.Body.Close()
		filePath := filepath.Join("image_found", fmt.Sprintf("%v.%v", index, i.Format))
		file, err := os.Create(filePath)
		if err != nil {
			continue
		}
		defer file.Close()
		_, err = io.Copy(file, response.Body)
		if err != nil {
			continue
		}

		fmt.Printf("Image saved to: %s\n", filePath)
	}
	return nil
}

func (cr *crawler) handleShow(writer http.ResponseWriter, request *http.Request) {
	time.Sleep(1 * time.Second)
	err := request.ParseForm()
	if err != nil {
		log.Printf("error parsing html form: %v", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	images, err := cr.app.GetAll()
	//SaveImagesToFileDirectory(images)
	sb_site := request.Form.Get("search_bar_site")
	sb_res := request.Form.Get("search_bar_res")
	if sb_res == "" && sb_site == "" {
	} else if sb_res != "" && sb_site != "" {
		images, err = cr.app.FindBySiteNameAndResolution(sb_site, sb_res)
	} else if sb_site != "" {
		images, err = cr.app.FindBySiteName(sb_site)
	} else if sb_res != "" {
		images, err = cr.app.FindByResolution(sb_res)
	}

	if err != nil {
		log.Printf("failed to get posts: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	cr.templateShow.Execute(writer, images)
}

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	crawlerRepo := mysql.New(mysql.MySQLOptions{
		URI: fmt.Sprintf("%s:%s@tcp(127.0.0.1)/img_crawler_app",
			"{name}", "{password}"),
	})
	crawlerRepo.Init()
	crawler_appp := crawler_app.New(crawlerRepo)
	app := crawler{
		server: server,
		mux:    mux,
		app:    crawler_appp,
	}

	//go run . -crawl=true https://en.wikipedia.org/wiki/Go_(programming_language) 5
	//go run . -crawl=false https://en.wikipedia.org/wiki/Go_(programming_language) 5
	//go run . -crawl=true https://en.wikipedia.org/wiki/Go_(programming_language) 10
	//go run . -crawl=false https://en.wikipedia.org/wiki/Go_(programming_language) 10

	MAX_GOROUTINES_COUNT = 6
	crawlptr := flag.Bool("crawl", false, "a bool")
	input := os.Args[2:]
	input_link = input[0]
	if len(input) > 1 {
		seconds, _ = strconv.Atoi(input[1])
		if seconds <= 0 {
			seconds = 120
		}
	} else {
		seconds = 120
	}
	flag.Parse()
	crawl_inner = *crawlptr
	app.Run()
}
