package main

import (
	"database/sql"
	// "fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"html/template"
	"log"
	"net/http"
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

type user struct {
	Email     string
	FirstName string
	LastName  string
}

var dbUsers = map[string]user{}      // email, user
var dbSessions = map[string]string{} // sessionid, email

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
	r.GET("/login", loginForm)
	r.POST("/login", authenticateUser)
	r.GET("/films/", Index)
	r.GET("/films/show/:id", Show)
	r.GET("/films/create", Create)
	r.POST("/films/store", Store)

	log.Fatal(http.ListenAndServe(":8000", r))
}

func loginForm(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// fmt.Print(dbUsers)
	// fmt.Print(dbSessions)
	// get cookie
	cookie, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
			//Secure: true,
			HttpOnly: true, // cannot access this cookie with Javascript
		}
		http.SetCookie(w, cookie)
	}

	// if user exists already, get user
	var u user
	sessionId := cookie.Value
	if email, ok := dbSessions[sessionId]; ok {
		u = dbUsers[email]
		http.Redirect(w, r, "/films", http.StatusSeeOther)
		return
	}

	tpl, err := template.ParseFiles("templates/auth/login.html")
	if err != nil {
		log.Fatalln("Error parsing template.", err)
	}
	tpl.ExecuteTemplate(w, "login.html", u)
}

func authenticateUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	cookie, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
			//Secure: true,
			HttpOnly: true, // cannot access this cookie with Javascript
		}
		http.SetCookie(w, cookie)
	}

	// if user is already logged in, redirect to default
	var u user
	sessionId := cookie.Value
	if _, ok := dbSessions[sessionId]; ok {
		http.Redirect(w, r, "/films", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	u, err = getUser(email)
	if err != nil {
		// fmt.Print("user was not found");
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	dbSessions[sessionId] = u.Email
	dbUsers[u.Email] = u

	http.Redirect(w, r, "/films", http.StatusSeeOther)
	return
}

func getUser(email string) (user, error) {
	stmt, err := db.Prepare("SELECT firstname, lastname, email FROM users WHERE email = ? LIMIT 1")
	defer stmt.Close()
	handleErr(err)

	var u user
	err = stmt.QueryRow(email).Scan(&u.FirstName, &u.LastName, &u.Email)
	if err != nil {
		return u, err
	}

	return u, nil
}

// Index, lists all the film entries
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// fmt.Print(dbUsers)
	// fmt.Print(dbSessions)

	cookie, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	// userId, ok := dbSessions[cookie.Value]
	_, ok := dbSessions[cookie.Value]

	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query("SELECT id, title, year, category FROM films")
	defer rows.Close()
	handleErr(err)

	films := []film{}

	for rows.Next() {
		var f film
		err = rows.Scan(&f.ID, &f.Title, &f.Year, &f.Category)
		handleErr(err)
		films = append(films, f)
	}
	handleErr(rows.Err())

	data := struct {
		Films []film
		Title string
	}{
		Films: films,
		Title: "List",
	}

	renderView(w, "templates/films/", "index.html", data)
	return
}

// Create, displays a html form for creating film entries
func Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	data := struct {
		Title string
	}{
		Title: "Add film",
	}

	renderView(w, "templates/films/", "create.html", data)

}

// Store, saves film entry to database
func Store(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// fmt.Println("title", r.FormValue("title"))

	if len(r.FormValue("title")) == 0 {
		//fmt.Fprintln(w, "title field is required.")
		flash = "the title is required"
		http.Redirect(w, r, "/films/create/", http.StatusSeeOther)
		return
	}

	//return
	stmt, err := db.Prepare("INSERT films SET title=?, year=?, category=?")
	handleErr(err)
	defer stmt.Close()

	result, err := stmt.Exec(r.FormValue("title"), "1981", "action")
	handleErr(err)

	// affectedRows, err := result.RowsAffected()
	_, err = result.RowsAffected()
	handleErr(err)

	// fmt.Fprintf(w, "Inserted %d record/s.", affectedRows)

	http.Redirect(w, r, "/films", http.StatusSeeOther)
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
		Film  film
		Title string
	}{
		Film:  f,
		Title: f.Title,
	}

	renderView(w, "templates/films/", "show.html", data)
	return
}

func renderView(w http.ResponseWriter, templatePath string, templateName string, data interface{}) {

	t := templatePath + templateName // todo check if templatePath ends with "/" and append it if it doesn't
	templates, err := template.ParseFiles(t, "templates/layout/head.html", "templates/layout/footer.html")
	if err != nil {
		log.Fatalln("Error parsing template.", err)
	}
	//tpl.ExecuteTemplate(w, "show.html", data)
	t1 := templates.Lookup("head.html")
	t1.ExecuteTemplate(w, "head", data)
	t2 := templates.Lookup(templateName)
	t2.ExecuteTemplate(w, "content", data)
	t3 := templates.Lookup("footer.html")
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
