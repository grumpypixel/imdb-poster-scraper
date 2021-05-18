package imdb

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/grumpypixel/go-utils/stringslice"
	"github.com/grumpypixel/go-webget"
)

const imdbBaseURL = "https://www.imdb.com/"

type PosterCollector struct {
	Posters []*Poster
	mutex   sync.Mutex
}

func (c *PosterCollector) Add(poster *Poster) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Posters = append(c.Posters, poster)
}

type Poster struct {
	MovieURL string
	ImageURL string
	Index    int
}

func (db *IMDB) collectPosters(movieList []string) []*Poster {
	collector := PosterCollector{}
	waitGroup := sync.WaitGroup{}
	for _, movie := range movieList {
		waitGroup.Add(1)
		go func(movie string) {
			movieURL, ok := db.validateMovieSource(movie)
			if ok {
				list, err := db.findPoster(movieURL)
				if err == nil {
					for i, imageURL := range list {
						collector.Add(&Poster{MovieURL: movieURL, ImageURL: imageURL, Index: i})
					}
				}
			}
			waitGroup.Done()
		}(movie)
		time.Sleep(db.WaitBetweenRequests)
		if db.Verbose {
			fmt.Printf(".")
		}
	}
	waitGroup.Wait()
	return collector.Posters
}

func (db *IMDB) findPoster(imdbURL string) ([]string, error) {
	mediaViewerURL, err := db.findMediaViewer(imdbURL)
	if err != nil {
		return nil, err
	}

	mediaViewerURL = imdbBaseURL + mediaViewerURL
	posters, err := db.findPostersInMediaViewer(mediaViewerURL)
	if err != nil {
		return nil, err
	}
	return posters, nil
}

func (db *IMDB) findMovieTitle(imdbURL string) (string, error) {
	response, err := http.Get(imdbURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return "", err
	}

	title := ""
	doc.Find("h1").Each(func(index int, element *goquery.Selection) {
		parent := element.Parent()
		if !parent.Is("div") || !parent.HasClass("title_wrapper") {
			return
		}
		text := element.Text()
		title = text
	})
	return strings.TrimSpace(title), nil
}

func (db *IMDB) findMediaViewer(imdbURL string) (string, error) {
	response, err := http.Get(imdbURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return "", err
	}

	mediaViewerURL := ""
	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		parent := element.Parent()
		if !parent.Is("div") || !parent.HasClass("poster") {
			return
		}
		href, exists := element.Attr("href")
		if exists && mediaViewerURL == "" {
			mediaViewerURL = cleanURL(href)
		}
	})
	if mediaViewerURL == "" {
		return "", fmt.Errorf(fmt.Sprintf("could not find mediaviewer: %s", imdbURL))
	}
	return mediaViewerURL, nil
}

func (db *IMDB) findPostersInMediaViewer(mediaViewerURL string) ([]string, error) {
	mediaID, ok := db.mediaIDFromMediaViewerURL(mediaViewerURL)
	if !ok {
		return nil, fmt.Errorf("could not retrieve media id: %s", mediaViewerURL)
	}

	response, err := http.Get(mediaViewerURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	posters := make([]string, 0)
	doc.Find("img").Each(func(index int, element *goquery.Selection) {
		parent := element.Parent()
		if !parent.Is("div") {
			return
		}
		_, exists := parent.Attr("class")
		if !exists {
			return
		}
		imageURL, exists := element.Attr("src")
		if !exists {
			return
		}
		dataImageAttr, exists := element.Attr("data-image-id")
		if !exists {
			return
		}
		if !strings.HasPrefix(dataImageAttr, mediaID) {
			return
		}
		posters = append(posters, imageURL)

		if !db.AllPosterResolutions {
			return
		}
		srcSet, exists := element.Attr("srcset")
		if !exists {
			return
		}
		images := strings.Split(srcSet, ",")
		for _, image := range images {
			s := strings.Split(image, " ")
			s = stringslice.TrimElements(s)
			s = stringslice.RemoveEmptyElements(s)
			if len(s) == 2 {
				imageURL := s[0]
				posters = append(posters, imageURL)
			}
		}
	})
	if len(posters) == 0 {
		return nil, fmt.Errorf(fmt.Sprintf("could not find any posters: %s", mediaViewerURL))
	}
	return posters, nil
}

func (db *IMDB) titleIDFromURL(imdbURL string) (string, bool) {
	if !strings.Contains(imdbURL, "title/tt") {
		return "", false
	}

	s := strings.Split(imdbURL, "/")
	s = stringslice.TrimElements(s)
	s = stringslice.RemoveEmptyElements(s)

	indexTitle := stringslice.IndexOfElement("title", s)
	if indexTitle == -1 || indexTitle == len(s)-1 {
		return "", false
	}
	title := s[indexTitle+1]
	return title, true
}

func (db *IMDB) mediaIDFromMediaViewerURL(mediaViewerURL string) (string, bool) {
	parts := strings.Split(mediaViewerURL, "/")
	parts = stringslice.TrimElements(parts)
	parts = stringslice.RemoveEmptyElements(parts)

	if len(parts) == 0 {
		return "", false
	}
	return parts[len(parts)-1], true
}

func (db *IMDB) validateURL(url string) (string, bool) {
	if strings.HasPrefix(url, "https://www.imdb.com") || strings.HasPrefix(url, "http://www.imdb.com") {
		return url, true
	} else if strings.HasPrefix(url, "www.imdb.com") {
		return "https://" + url, true
	}
	return url, false
}

func (db *IMDB) validateMovieSource(src string) (string, bool) {
	src = cleanURL(src)
	if strings.HasPrefix(src, "https://www.imdb.com/title/") || strings.HasPrefix(src, "http://www.imdb.com/title/") {
		return src, true
	} else if strings.HasPrefix(src, "www.imdb.com/title/") {
		return "https://" + src, true
	} else if strings.HasPrefix(src, "imdb.com/title/") {
		return "https://www." + src, true
	} else if strings.HasPrefix(src, "/title/tt") {
		return "https://www.imdb.com" + src, true
	} else if strings.HasPrefix(src, "title/tt") {
		return "https://www.imdb.com/" + src, true
	} else if db.isTitleID(src) {
		return db.makeURLFromTitleID(src), true
	}
	return src, false
}

func (db *IMDB) makeURLFromTitleID(titleID string) string {
	return imdbBaseURL + "title/" + titleID + "/"
}

func (db *IMDB) isTitleID(titleID string) bool {
	t := strings.ToLower(titleID)
	if !strings.HasPrefix(t, "tt") {
		return false
	}
	return containsDigitsOnly(titleID[2:])
}

func containsDigitsOnly(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func cleanURL(url string) string {
	index := strings.Index(url, "?")
	if index >= 0 {
		url = url[:index]
	}
	return strings.TrimSpace(url)
}

func download(url string, targetDir, filename string, waitGroup *sync.WaitGroup, progress webget.ProgressHandler) error {
	options := &webget.Options{
		ProgressHandler: progress,
		Timeout:         time.Second * 60,
		CreateTargetDir: true,
	}

	if err := webget.DownloadToFile(url, targetDir, filename, options); err != nil {
		return err
	}

	waitGroup.Done()
	return nil
}

// Something for the attic. For the moment at least.
// var scrapeList StringList
// flag.Var(&scrapeList, "s", "Scrape all movie links from an IMDB URL")
// if len(scrapeList) > 0 {
// 	fmt.Println("Scraping...")
// 	for _, url := range scrapeList {
// 		links, err := imdb.ScrapeMovieLinks(url)
// 		if err != nil {
// 			fmt.Println(err)
// 			continue
// 		}
// 		if len(links) > 0 {
// 			movieList = append(movieList, links...)
// 		}
// 		time.Sleep(imdb.WaitBetweenRequests)
// 	}
// }
// func (db *IMDB) ScrapeMovieLinks(imdbURL string) ([]string, error) {
// 	url, ok := db.validateURL(imdbURL)
// 	if !ok {
// 		fmt.Println("Invalid IMDB URL:", url)
// 	}

// 	response, err := http.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer response.Body.Close()

// 	doc, err := goquery.NewDocumentFromReader(response.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var links []string
// 	doc.Find("a").Each(func(index int, element *goquery.Selection) {
// 		href, exists := element.Attr("href")
// 		if !exists {
// 			return
// 		}
// 		url := cleanURL(href)
// 		if !strings.Contains(url, "title/tt") {
// 			return
// 		}
// 		url, ok := db.validateMovieSource(url)
// 		if !ok {
// 			fmt.Println("Invalid source:", url)
// 			return
// 		}
// 		links = append(links, url)
// 	})
// 	return links, nil
// }
