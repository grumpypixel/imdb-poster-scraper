# imdb-poster-scraper

A movie poster downloader that finds and scrapes posters from IMDB. Nothing more, nothing less.

## Before you start to go nuts with this

Please note that IMDB will *block* your IP for a certain time if you overdo it. Please use with care and be respectful with their data and resources. Thanks.

## How to run without building

```console
$ go run main.go [options]
```

## How to build

```console
$ go build -o imdb-poster-scraper main.go
```

## Command-line options

* --m *IMDB_MOVIE_TITLE_ID_OR_URL* (Specify the movie title ID or the URL of the movie poster you want to be downloaded)
* --dir *DIRECTORY* (Specify the target directory where the downloaded posters should be saved to)
* --wait *NUMBER* (Wait for *NUMBER* of milliseconds between requests)
* --all (Download all poster resolutions, not just the biggest one)
* --collect (Collect posters only, don't download anything)
* --id (Search for the movie title and the movie's title identifier)
* --silent (Stealth mode, don't make a sound)

## Examples

```
# Download posters by specifying movie title IDs
$ go run main.go -m tt2861424 -m tt0149460
```

```
# Mix movie title IDs and URLs
$ go run main.go -m tt2861424 -m www.imdb.com/title/tt0149460/ -m https://www.imdb.com/title/tt0245429/
```

```
# Specify a target directory where the posters should be saved to
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429
```

```
# Download posters in all available resolutions
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429 --all
```

```
# Wait for 1 second between downloads
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429 --all --wait 1000
```

```
# Collect all poster URLs but do not download. This probably is not really helpful, so just ignore it.
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429 --all --collect
```

```
# List movie title IDs and movie titles (just for fun)
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429 --list
```

```
# Be stealthy, don't say a word, don't make sound.
$ go run main.go --dir ./posters -m tt2861424 -m tt0149460 -m tt0245429 --silent
```
