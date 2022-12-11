package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileItem struct {
	Name  string
	Size  string
	Date  string
	IsDir bool
	Path  string
}

type templateVariables struct {
	Path  []breadcrumb
	Files []fileItem
}

type breadcrumb struct {
	Name string
	Path string
}

var dataDir string = "/"

func Start() {
	http.ListenAndServe(fmt.Sprintf(":8083"), Handler{})
}

type Handler struct {
}

func (a Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Filepath, from the root data dir
	fp := filepath.Join(dataDir, filepath.Clean(r.URL.Path))
	// Cleaned filepath, without the root data dir, used for template rendering purpose
	cfp := strings.Replace(fp, dataDir, "", 1)

	// Return a 404 if the template doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			notFound, _ := template.ParseFS(NotFound, "404.html")
			w.WriteHeader(http.StatusNotFound)
			notFound.ExecuteTemplate(w, "404.html", nil)
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		fmt.Println("name: ", info.Name(), " size:", humanize.Bytes(uint64(info.Size())), " modTime: ", humanize.Time(info.ModTime()))
		files, err := os.ReadDir(fp)
		if err != nil {
			log.Error().Err(err)
		}

		// Init template variables
		templateVars := templateVariables{}

		// Construct the breadcrumb
		path := strings.Split(cfp, "/")
		for len(path) > 1 {
			b := breadcrumb{
				Name: path[len(path)-1],
				Path: strings.Join(path, "/"),
			}
			path = path[:len(path)-1]
			templateVars.Path = append(templateVars.Path, b)
		}
		// Since the breadcrumb built is not very ordered...
		// REVERSE ALL THE THINGS
		for left, right := 0, len(templateVars.Path)-1; left < right; left, right = left+1, right-1 {
			templateVars.Path[left], templateVars.Path[right] = templateVars.Path[right], templateVars.Path[left]
		}

		// Establish list of files in the current directory
		for _, f := range files {
			if !strings.HasPrefix(f.Name(), ".") {
				inf, err := f.Info()
				var size int64
				var modTime time.Time
				if err == nil {
					size = inf.Size()
					modTime = inf.ModTime()
				}
				p := filepath.Join(cfp, filepath.Clean(f.Name()))
				p = filepath.ToSlash(p)
				templateVars.Files = append(templateVars.Files, fileItem{
					Name:  f.Name(),
					Size:  humanize.Bytes(uint64(size)),
					Date:  humanize.Time(modTime),
					IsDir: f.IsDir(),
					Path:  p,
				})
			}
		}

		// Prepare the template
		tmpl, err := template.ParseFS(Index, "index.html")
		if err != nil {
			// Log the detailed error
			log.Error().Err(err)
			// Return a generic "Internal Server Error" message
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Return file listing in the template
		if err := tmpl.ExecuteTemplate(w, "index.html", templateVars); err != nil {
			log.Error().Err(err)
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}

	if !info.IsDir() {
		content, _ := os.Open(fp)
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
		http.ServeContent(w, r, fp, info.ModTime(), content)
		return
	}
}
