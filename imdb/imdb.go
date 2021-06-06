package imdb

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/grumpypixel/go-webget"
)

type IMDB struct {
	AllPosterResolutions bool
	WaitBetweenRequests  time.Duration
	Verbose              bool
}

type PosterProgress struct {
	Verbose bool
}

func (p PosterProgress) Start(sourceURL string) {}

func (p PosterProgress) Update(sourceURL string, percentage float64, bytesRead, contentLength int64) {
	if p.Verbose {
		fmt.Printf(".")
	}
}

func (p PosterProgress) Done(sourceURL string) {
	if p.Verbose {
		fmt.Printf("\\o/")
	}
}

func (db *IMDB) DownloadPosters(movieList []string, targetDir string, progress webget.ProgressHandler) {
	posters, errs := db.collectPosters(movieList)

	errors := ErrorCollector{}
	if len(errs) > 0 {
		errors.Errors = append(errors.Errors, errs...)
	}

	waitGroup := sync.WaitGroup{}
	for _, poster := range posters {
		waitGroup.Add(1)
		go func(poster *Poster) {
			url := poster.ImageURL
			titleID, _ := db.titleIDFromURL(poster.MovieURL)
			extension := filepath.Ext(url)
			filename := fmt.Sprintf("%s-%.2d%s", titleID, poster.Index, extension)

			err := download(url, targetDir, filename, &waitGroup, progress)
			if err != nil {
				errors.Add(err)
			}
		}(poster)
		time.Sleep(db.WaitBetweenRequests)
	}
	waitGroup.Wait()

	if db.Verbose {
		fmt.Println()
		for _, err := range errors.Errors {
			fmt.Println(err)
		}
	}
}

func (db *IMDB) FindPosters(movieList []string) []string {
	posters, _ := db.collectPosters(movieList)
	var urls []string
	for _, poster := range posters {
		urls = append(urls, poster.ImageURL)
	}
	return urls
}

// returns title ID and title
func (db *IMDB) GetMovieTitle(imdbURL string) (string, string) {
	url, ok := db.validateMovieSource(imdbURL)
	if !ok {
		return "", ""
	}
	titleID, _ := db.titleIDFromURL(url)
	title, _ := db.findMovieTitle(url)
	return titleID, title
}
