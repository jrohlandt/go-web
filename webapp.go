package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

// index: list of movies

// create: form to create movies

// store: process form data and save to db

// show: page to show a movie details

var db *sql.DB
var err error
var flash string

type film struct {
	ID       int
	Title    string
	Year     string
	Rating   float64
	Category string
}

func main() {
	db, err = sql.Open("mysql", "root:xocjm@tcp(127.0.0.1:3306)/go_local?charset=utf8")
	handleErr(err)
	defer db.Close()

	err = db.Ping()
	handleErr(err)

	// todo handle favicon and static files with httprouter?
	//http.Handle("/resources/", http.StripPrefix("/resources", http.FileServer(http.Dir("./assets"))))
	//http.HandleFunc("/favicon.ico", faviconHandler)

	r := httprouter.New()
	r.GET("/films/", Index)
	r.GET("/films/show/:id", Show)
	r.GET("/films/create", Create)
	r.POST("/films/store", Store)

	log.Fatal(http.ListenAndServe(":8000", r))
}

// Index, lists all the film entries
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	rows, err := db.Query("SELECT id, title, year, category FROM films")
	handleErr(err)

	films := []film{}

	for rows.Next() {
		var f film
		err = rows.Scan(&f.ID, &f.Title, &f.Year, &f.Category)
		handleErr(err)
		films = append(films, f)
	}

	data := struct {
		Films []film
		Title string
	}{
		Films: films,
		Title: "List",
	}

	renderView(w, "templates/films/", "index.gohtml", data)
	return
}

// Create, displays a html form for creating film entries
func Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	data := struct {
		Title string
	}{
		Title: "Add film",
	}

	renderView(w, "templates/films/", "create.gohtml", data)

}

// Store, saves film entry to database
func Store(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	fmt.Println("title", r.FormValue("title"))

	if len(r.FormValue("title")) == 0 {
		//fmt.Fprintln(w, "title field is required.")
		flash = "the title is required"
		http.Redirect(w, r,"/films/create/", http.StatusSeeOther)
		return
	}

	//return
	stmt, err := db.Prepare("INSERT films SET title=?, year=?, category=?")
	handleErr(err)
	defer stmt.Close()

	result, err := stmt.Exec(r.FormValue("title"), "1981", "action")
	handleErr(err)

	affectedRows, err := result.RowsAffected()
	handleErr(err)

	fmt.Fprintf(w, "Inserted %d record/s.", affectedRows)

	http.Redirect(w, r, "/films/", http.StatusSeeOther)
	return
}

// Show, shows the details of a film entry
func Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	stmt, err := db.Prepare("SELECT id, title, year, category FROM films WHERE id=? ")
	handleErr(err)
	defer stmt.Close()

	row := stmt.QueryRow(params.ByName("id"))

	var f film
	err = row.Scan(&f.ID, &f.Title, &f.Year, &f.Category)

	data := struct {
		Film film
		Title string
	}{
		Film: f,
		Title: f.Title,
	}

	renderView(w, "templates/films/", "show.gohtml", data)
	return
}

func renderView(w http.ResponseWriter, templatePath string, templateName string, data interface{}) {

	t := templatePath + templateName // todo check if templatePath ends with "/" and append it if it doesn't
	templates, err := template.ParseFiles(t, "templates/layout/head.gohtml", "templates/layout/footer.gohtml")
	if err != nil {
		log.Fatalln("Error parsing template.", err)
	}
	//tpl.ExecuteTemplate(w, "show.gohtml", data)
	t1 := templates.Lookup("head.gohtml")
	t1.ExecuteTemplate(w, "head", data)
	t2 := templates.Lookup(templateName)
	t2.ExecuteTemplate(w, "content", data)
	t3 := templates.Lookup("footer.gohtml")
	t3.ExecuteTemplate(w, "footer", data)
	return
}

func Hello(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Hello, %s!\n", params.ByName("name"))
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./favicon.ico")
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
