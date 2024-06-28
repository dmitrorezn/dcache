package renderer

import (
	"github.com/labstack/echo/v4"
	"html/template"

	"io"
	"io/fs"
)

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer(pattern string) *Renderer {
	return &Renderer{
		tmpl: template.Must(template.ParseGlob(pattern)),
	}
}

func NewRendererFS(fs fs.FS) *Renderer {
	return &Renderer{
		tmpl: template.Must(template.ParseFS(fs, "*")),
	}
}

func (r Renderer) Render(writer io.Writer, name string, data interface{}, _ echo.Context) error {
	return r.tmpl.ExecuteTemplate(writer, name+".html", data)
}

var _ echo.Renderer = Renderer{}
