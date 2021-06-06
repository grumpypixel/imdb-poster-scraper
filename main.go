package main

import (
	"app/imdb"
	"flag"
	"fmt"
	"sync"
	"time"
)

type StringList []string

func (i *StringList) String() string {
	return "StringList"
}

func (i *StringList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var movieList StringList

	flag.Var(&movieList, "m", "Movie title ID (e.g.: tt2861424) or IMDB URL (e.g.: www.imdb.com/title/tt0149460/")
	targetDir := flag.String("dir", "./", "Target directory")
	wait := flag.Int("wait", 0, "Wait for n milliseconds between requests")
	all := flag.Bool("all", false, "Download all poster resolutions")
	collect := flag.Bool("collect", false, "Don't download posters. Collect only.")
	listTitles := flag.Bool("list", false, "List movie titles before doing anything else")
	silent := flag.Bool("silent", false, "Speak nothing, friend, and do not enter.")
	flag.Parse()

	beVerbose := !*silent

	db := imdb.IMDB{
		AllPosterResolutions: *all,
		WaitBetweenRequests:  time.Millisecond * time.Duration(*wait),
		Verbose:              beVerbose,
	}

	blab := &Blabber{Verbose: !*silent}
	if len(movieList) == 0 {
		blab.Println("Nothing to do. Bye.")
		return
	}

	if *listTitles && beVerbose {
		blab.Println("Listing movies")
		waitGroup := &sync.WaitGroup{}
		for _, movie := range movieList {
			waitGroup.Add(1)
			go func(movie string) {
				fmt.Println(db.GetMovieTitle(movie))
				waitGroup.Done()
			}(movie)
			time.Sleep(db.WaitBetweenRequests)
		}
		waitGroup.Wait()
		blab.Println()
	}

	if *collect {
		blab.Printf("Collecting posters")
		posters := db.FindPosters(movieList)
		blab.Println()
		for i, poster := range posters {
			blab.Println(fmt.Sprintf("#%d: %s", i+1, poster))
		}
		blab.Println()
	} else {
		blab.Printf("Downloading posters")
		db.DownloadPosters(movieList, *targetDir, &imdb.PosterProgress{Verbose: beVerbose})
		blab.Println()
	}
	blab.Println("Done.")
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
