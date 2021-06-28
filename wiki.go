package main

import (
	"html/template"
	"io/ioutil"
    "log"
    "net/http"
    "regexp"
)


type Page struct {
	Title string
	Body []byte
}

func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

var templates = template.Must(template.New("").Funcs(template.FuncMap{
    "htmlescaper": func(b []byte) template.HTML {
        return template.HTML(b)
    },
}).ParseFiles(
    "tmpl/root.html",
    "tmpl/edit.html",
    "tmpl/view.html",
))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    err := templates.ExecuteTemplate(w, tmpl + ".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

//////////////
// HANDLERS //
//////////////

func rootHandler(w http.ResponseWriter, r *http.Request) {
    files, err := ioutil.ReadDir("data/")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    titles := make([]string, len(files))
    for i, f := range files {
        name := f.Name()
        titles[i] = name[:len(name) - len(".txt")]
    }
    err = templates.ExecuteTemplate(w, "root.html", titles)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func gotoHandler(w http.ResponseWriter, r *http.Request) {
    title := r.FormValue("title")
    http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

var linkStub = regexp.MustCompile(`\[(\w+)\]`)

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        http.Redirect(w, r, "/edit/" + title, http.StatusFound)
        return
    }
    p.Body = linkStub.ReplaceAllFunc(p.Body, func(m []byte) []byte {
        title := linkStub.ReplaceAllString(string(m), `$1`)
        tag := "<a href='/view/" + title + "'>" + title + "</a>"
        return []byte(tag)
    })
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

var validPath = regexp.MustCompile(`^/(edit|save|view)/(\w+)$`)

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        fn(w, r, m[2])
    }
}

func main() {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/goto/", gotoHandler)
    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
