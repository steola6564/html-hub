package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed index.html
var indexHTML string

var index = template.Must(template.New("index").Parse(indexHTML))

const (
	pagesDir   = "pages"
	configFile = "config.conf"
)

type entry struct {
	Name, Label, URL string
}

func pageName(name string) bool {
	if strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\") {
		return false
	}
	ext := filepath.Ext(name)
	return strings.EqualFold(ext, ".html") || strings.EqualFold(ext, ".htm")
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(pagesDir) // ReadDir returns entries sorted by filename.
	if err != nil {
		log.Printf("read %s: %v", pagesDir, err)
		http.Error(w, "Cannot read catalogue", http.StatusInternalServerError)
		return
	}
	var pages []entry
	for _, f := range files {
		name := f.Name()
		if !f.Type().IsRegular() || !pageName(name) {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		pages = append(pages, entry{
			Name:  name,
			Label: strings.ReplaceAll(stem, "_", " "),
			URL:   "/pages/" + url.PathEscape(name),
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != http.MethodHead {
		if err := index.Execute(w, pages); err != nil {
			log.Printf("render catalogue: %v", err)
		}
	}
}

func servePage(w http.ResponseWriter, r *http.Request, name string) {
	if !pageName(name) {
		http.NotFound(w, r)
		return
	}
	// On Linux these flags refuse symlinks and avoid blocking on a named pipe.
	f, err := os.OpenFile(filepath.Join(pagesDir, name), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeContent(w, r, name, info.ModTime(), f)
}

func route(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	switch {
	case r.URL.Path == "/":
		serveIndex(w, r)
	case strings.HasPrefix(r.URL.Path, "/pages/"):
		servePage(w, r, strings.TrimPrefix(r.URL.Path, "/pages/"))
	default:
		http.NotFound(w, r)
	}
}

func configuredAddr() (string, error) {
	// Only seed a missing file. Existing user settings belong to the operator.
	f, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err == nil {
		_, writeErr := f.WriteString("port=8090\n")
		closeErr := f.Close()
		if writeErr != nil {
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	} else if !os.IsExist(err) {
		return "", err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != "port" {
		return "", fmt.Errorf("%s: expected one setting, e.g. port=8090", configFile)
	}
	port, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("%s: port must be an integer from 1 to 65535", configFile)
	}
	return fmt.Sprintf(":%d", port), nil
}

func main() {
	addr := flag.String("addr", "", "override the configured HTTP listen address (host:port)")
	flag.Parse()
	if *addr == "" {
		configured, err := configuredAddr()
		if err != nil {
			log.Fatal(err)
		}
		*addr = configured
	}
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              *addr,
		Handler:           http.HandlerFunc(route),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("HTML catalogue: listening on %s, serving ./%s/", *addr, pagesDir)
	log.Fatal(server.ListenAndServe())
}
