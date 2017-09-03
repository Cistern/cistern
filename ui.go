package main

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Preetam/siesta"
)

func UI(path string) (http.Handler, error) {
	var err error
	templ, err := template.ParseGlob(filepath.Join(filepath.Join(path, "templates"), "*"))
	if err != nil {
		return nil, err
	}

	service := siesta.NewService("/ui/")
	//service.DisableTrimSlash()
	service.Route("GET", "/", "Index", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "index", map[string]string{})
	})
	service.SetNotFound(http.StripPrefix("/ui", http.FileServer(http.Dir(path))))

	return service, nil
}
