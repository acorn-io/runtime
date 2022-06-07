package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
)

var (
	//go:embed static/dashboard/*
	Files    embed.FS
	fsServer http.Handler
)

func init() {
	subFS, err := fs.Sub(Files, "static")
	if err != nil {
		panic(err)
	}
	fsServer = http.FileServer(http.FS(subFS))
}

func exists(path string) bool {
	stat, err := fs.Stat(Files, filepath.Join("static", path))
	return err == nil && !stat.IsDir()
}

func NotFound(indexURL string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if exists(req.URL.Path) {
			fsServer.ServeHTTP(rw, req)
		} else if exists("dashboard/index.html") {
			req.URL.Path = "dashboard/"
			req.URL.RawPath = "dashboard/"
			fsServer.ServeHTTP(rw, req)
		} else {
			resp, err := http.Get(indexURL)
			if err == nil {
				_, _ = io.Copy(rw, resp.Body)
				_ = resp.Body.Close()
			} else {
				http.Error(rw, err.Error(), resp.StatusCode)
			}
		}
	})
}
