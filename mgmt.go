package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
)

func (a *App) addMgmt(r *mux.Router) {
	r.HandleFunc("/mgmt", basicAuth(a.indexHandler)).Methods("GET")
	r.HandleFunc("/mgmt/add", basicAuth(a.addUserHandler)).Methods("POST")
	r.HandleFunc("/mgmt/del", basicAuth(a.delUserHandler)).Methods("POST")
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
	t := template.Must(template.New("main").Parse(bodyTemplate))
	t.New("body").Parse(indexTemplate)

	type lfs struct {
		Users []*MetaUser
	}

	users, err := a.metaStore.Users()
	if err != nil {
		fmt.Fprintf(w, "Error retrieving users: %s", err)
		return
	}

	l := &lfs{Users: users}

	t.Execute(w, l)
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

	http.Redirect(w, r, "/mgmt", 302)
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

	http.Redirect(w, r, "/mgmt", 302)
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

var bodyTemplate = `<html>
<head>
	<title>Git LFS Server Management</title>
</head>
<body>
{{template "body" .}}
</body>
</html>
`

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
