package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grumpypixel/go-webget"
	imdb "github.com/grumpypixel/imdb-poster-go"
)

func main() {
	var movies StringList

	flag.Var(&movies, "m", "The movie's title ID (e.g.: tt2861424) or an IMDB URL (e.g.: www.imdb.com/title/tt0149460/")
	targetDir := flag.String("dir", "./", "Specify a target directory (will be created if it does not exist)")
	delay := flag.Int("delay", 0, "Delay between requests in milliseconds")
	allResolutions := flag.Bool("all", false, "Download all poster resolutions")
	collect := flag.Bool("collect", false, "Don't download posters. Collect URLs only.")
	listTitles := flag.Bool("list", false, "List movie titles before doing anything else")
	silent := flag.Bool("shhh", false, "Speak nothing, friend, and do not enter.")
	flag.Parse()

	delayBetweenRequests := time.Millisecond * time.Duration(*delay)

	db := imdb.NewIMDB(*allResolutions)

	blab := Blabber{Verbose: !*silent}
	if len(movies) == 0 {
		blab.Println("Nothing to do. Bye.")
		return
	}

	if *listTitles && !*silent {
		waitGroup := &sync.WaitGroup{}
		for _, movie := range movies {
			waitGroup.Add(1)
			go func(movie string) {
				fmt.Println(db.FetchTitle(movie))
				waitGroup.Done()
			}(movie)
			time.Sleep(delayBetweenRequests)
		}
		waitGroup.Wait()
	}

	blab.Printf("Fetching...")
	posters := fetchPosters(db, movies, &blab)

	if *collect {
		blab.Println()
		for _, poster := range posters {
			for i, url := range poster.Images {
				blab.Println(fmt.Sprintf("#%d: %s", i+1, url))
			}
		}
		blab.Println()
	} else {
		ok := true
		if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
			err := os.MkdirAll(*targetDir, os.ModePerm)
			if err != nil {
				fmt.Println("\nCould not create target directory")
				ok = false
			}
		}
		if ok {
			downloadPosters(posters, *targetDir, delayBetweenRequests, &blab)
		}
		blab.Println()
	}
	blab.Println("Done.")
}

type StringList []string

func (i *StringList) String() string {
	return "StringList"
}

func (i *StringList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Blabber struct {
	Verbose bool
}

func (b Blabber) Println(a ...interface{}) {
	if b.Verbose {
		fmt.Println(a...)
	}
}

func (b Blabber) Printf(format string, a ...interface{}) {
	if b.Verbose {
		fmt.Printf(format, a...)
	}
}

type Progress struct {
	Blabber *Blabber
}

func (p Progress) Start(sourceURL string) {}

func (p Progress) Update(sourceURL string, percentage float64, bytesRead, contentLength int64) {
	p.Blabber.Printf(".")
}

func (p Progress) Done(sourceURL string) {
	p.Blabber.Printf("\\o/")
}

type MoviePoster struct {
	Source   string
	TitleID  string
	Title    string
	MovieURL string
	Images   []string
}

func fetchPosters(db *imdb.IMDB, movies []string, blab *Blabber) []MoviePoster {
	moviePosters := []MoviePoster{}
	for _, movie := range movies {
		url, ok := db.URLFromSource(movie)
		if !ok {
			continue
		}
		id, ok := db.TitleIDFromURL(url)
		if !ok {
			continue
		}
		moviePoster := MoviePoster{
			Source:   movie,
			TitleID:  id,
			MovieURL: url,
			Images:   []string{},
		}
		posterURLs := db.FetchPoster(movie)
		if len(posterURLs) > 0 {
			moviePoster.Images = append(moviePoster.Images, posterURLs...)
			moviePosters = append(moviePosters, moviePoster)
		}
		blab.Printf(".")
	}
	return moviePosters
}

func downloadPosters(posters []MoviePoster, targetDir string, delay time.Duration, blab *Blabber) {
	progress := &Progress{Blabber: blab}
	waitGroup := sync.WaitGroup{}
	for _, poster := range posters {
		for i, url := range poster.Images {
			ext := filepath.Ext(url)
			filename := fmt.Sprintf("%s-%.2d%s", poster.TitleID, i, ext)
			waitGroup.Add(1)
			go download(url, targetDir, filename, &waitGroup, progress)
			time.Sleep(delay)
		}
	}
	waitGroup.Wait()
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
