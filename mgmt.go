package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
)

var (
	cssBox      *rice.Box
	templateBox *rice.Box
)

type pageData struct {
	Name   string
	Config *Configuration
	Users  []*MetaUser
}

func (a *App) addMgmt(r *mux.Router) {
	r.HandleFunc("/mgmt", basicAuth(a.indexHandler)).Methods("GET")
	r.HandleFunc("/mgmt/users", basicAuth(a.usersHandler)).Methods("GET")
	r.HandleFunc("/mgmt/add", basicAuth(a.addUserHandler)).Methods("POST")
	r.HandleFunc("/mgmt/del", basicAuth(a.delUserHandler)).Methods("POST")

	cssBox = rice.MustFindBox("mgmt/css")
	templateBox = rice.MustFindBox("mgmt/templates")
	r.HandleFunc("/mgmt/css/{file}", cssHandler)
}

func cssHandler(w http.ResponseWriter, r *http.Request) {
	file := mux.Vars(r)["file"]
	f, err := cssBox.Open(file)
	if err != nil {
		http.Error(w, "Not Found", 404)
		logRequest(r, 404)
		return
	}

	w.Header().Set("Content-Type", "text/css")

	io.Copy(w, f)
	f.Close()
}

func basicAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Config.AdminUser == "" || Config.AdminPass == "" {
			http.Error(w, "Not Found", 404)
			logRequest(r, 404)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=mgmt")
			http.Error(w, "authorization failed", 401)
			logRequest(r, 401)
			return
		}

		if user != Config.AdminUser || pass != Config.AdminPass {
			w.Header().Set("WWW-Authenticate", "Basic realm=mgmt")
			http.Error(w, "authorization failed", 401)
			logRequest(r, 401)
			return
		}

		h(w, r)
		logRequest(r, 200)
	}
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	bodyString, err := templateBox.String("body.tmpl")
	if err != nil {
		http.Error(w, "Not Found", 404)
		logRequest(r, 404)
		return
	}

	configString, err := templateBox.String("config.tmpl")
	if err != nil {
		http.Error(w, "Not Found", 404)
		logRequest(r, 404)
		return
	}

	t := template.Must(template.New("main").Parse(bodyString))
	t.New("content").Parse(configString)

	t.Execute(w, pageData{Name: "index", Config: Config})
}

func (a *App) usersHandler(w http.ResponseWriter, r *http.Request) {
	bodyString, err := templateBox.String("body.tmpl")
	if err != nil {
		http.Error(w, "Not Found", 404)
		logRequest(r, 404)
		return
	}

	contentString, err := templateBox.String("users.tmpl")
	if err != nil {
		http.Error(w, "Not Found", 404)
		logRequest(r, 404)
		return
	}

	t := template.Must(template.New("main").Parse(bodyString))
	t.New("content").Parse(contentString)

	users, err := a.metaStore.Users()
	if err != nil {
		fmt.Fprintf(w, "Error retrieving users: %s", err)
		return
	}

	t.Execute(w, pageData{Name: "users", Users: users})
}

func (a *App) addUserHandler(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("name")
	pass := r.FormValue("password")
	if user == "" || pass == "" {
		fmt.Fprintf(w, "Invalid username or password")
		return
	}

	if err := a.metaStore.AddUser(user, pass); err != nil {
		fmt.Fprintf(w, "Error adding user: %s", err)
		return
	}

	http.Redirect(w, r, "/mgmt/users", 302)
}

func (a *App) delUserHandler(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("name")
	if user == "" {
		fmt.Fprintf(w, "Invalid username")
		return
	}

	if err := a.metaStore.DeleteUser(user); err != nil {
		fmt.Fprintf(w, "Error deleting user: %s", err)
		return
	}

	http.Redirect(w, r, "/mgmt/users", 302)
}

func authenticate(r *http.Request) error {
	err := errors.New("Forbidden")

	if Config.AdminUser == "" || Config.AdminPass == "" {
		return err
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		return err
	}

	if user == Config.AdminUser && pass == Config.AdminPass {
		return nil
	}
	return err
}

var indexTemplate = `
<h2>Users</h2>
{{range .Users}}
<div>{{.Name}} <form method="POST" action="/mgmt/del"><input type="hidden" name="name" value="{{.Name}}"/><input type="submit" value="Delete"/></form></div>
{{end}}

<form method="POST" action="/mgmt/add">
<label id="name">Name:</label>
<input type="text" name="name" />
<input type="password" name="password" />
<input type="submit" value="Add User" />
</form>
`
