package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	// Rutas relativas a los directorios de templates y estilos
	// (Se asume que el comando se ejecuta desde la raíz del proyecto autoMarket_AAA_workshop)
	templatesDir := filepath.Join("web", "templates")
	stylesDir := filepath.Join(templatesDir, "styles")

	// Parsear los templates .html dentro del directorio de templates
	tmpl := template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html")))

	// Habilitar el servidor de archivos estáticos para compartir el CSS
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir(stylesDir))))

	// Ruta para el Login (/login)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Simulamos un login exitoso redirigiendo a la pantalla principal
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		err := tmpl.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Ruta principal para el Marketplace / Catálogo (/)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		err := tmpl.ExecuteTemplate(w, "index.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Iniciar el servidor
	port := ":8080"
	log.Printf("💻 Servidor AutoMarket iniciando en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
