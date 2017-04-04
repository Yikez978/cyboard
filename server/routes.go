package server

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// Logger is used to send logging messages to stdout.
var Logger = log.New(os.Stdout, " ", log.Ldate|log.Ltime|log.Lshortfile)

type Page struct {
	Title string
	T     Team
}

var templates map[string]*template.Template

// Parse templates at startup
// TODO Loop through all templates in directory
func init() {
	templates = make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"title":           strings.Title,
		"totalChallenges": getChallenges,
		"teamChallenges":  getTeamChallenges,
		"teamScore":       getTeamScore,
		"allTeamScores":   getAllTeamScores,
		"getStatus":       DataGetResultByService,
		"serviceList":     DataGetServiceList,
	}

	includes := mustGlobFiles("tmpl/includes/*.tmpl")
	layouts := mustGlobFiles("tmpl/*.tmpl")

	for _, layout := range layouts {
		files := append(includes, layout)
		title := strings.TrimSuffix(filepath.Base(layout), ".tmpl")
		templates[title] = template.Must(template.New(layout).Funcs(funcMap).ParseFiles(files...))
	}
}

func CreateWebRouter() *mux.Router {
	router := mux.NewRouter()
	// Public Routes
	router.HandleFunc("/", ShowHome).Methods("GET")
	router.HandleFunc("/login", ShowLogin).Methods("GET")
	router.HandleFunc("/login", SubmitLogin).Methods("POST")
	router.HandleFunc("/logout", Logout).Methods("GET")
	router.HandleFunc("/scoreboard", ShowScoreboard).Methods("GET")
	router.HandleFunc("/team/services", ShowServices).Methods("GET")
	// Public API
	// TODO: Make this the name of AIS challenge
	router.HandleFunc("/team/scores", GetScores).Methods("GET")
	router.HandleFunc("/team/scores/live", ServeScoresWs).Methods("GET")
	router.HandleFunc("/team/services/live", ServeServicesWs).Methods("GET")
	return router
}

func CreateTeamRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/team/dashboard", ShowTeamDashboard).Methods("GET")
	router.HandleFunc("/challenges", ShowChallenges).Methods("GET")
	router.HandleFunc("/challenges/list", GetChallenges).Methods("GET")
	router.HandleFunc("/challenges/verify", CheckFlag).Methods("POST")
	router.HandleFunc("/challenges/verify/all", CheckAllFlags).Methods("POST")
	return router
}

func ShowHome(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team")
	p := Page{Title: "homepage"}
	if t != nil {
		p.T = t.(Team)
	}
	renderTemplate(w, p)
}

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if context.Get(r, "team") == nil {
		p := Page{
			Title: "login",
		}
		renderTemplate(w, p)
	} else {
		http.Redirect(w, r, "/team/dashboard", 302)
	}
}

func SubmitLogin(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		Logger.Println("Getting from Store faile", err)
	}

	succ := CheckCreds(w, r)
	if succ {
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.Redirect(w, r, "/team/dashboard", 302)
		return
	}
	http.Redirect(w, r, "/login", 302)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	delete(session.Values, "id")
	// Make sure we save the session after deleting the ID.
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	http.Redirect(w, r, "/login", 302)
}

func ShowTeamDashboard(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Title: "dashboard",
		T:     context.Get(r, "team").(Team),
	}
	renderTemplate(w, p)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team")
	if t != nil {
		p := Page{
			Title: "challenges",
			T:     t.(Team),
		}
		renderTemplate(w, p)
	}
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team")
	p := Page{Title: "scoreboard"}
	if t != nil {
		p.T = t.(Team)
	}
	renderTemplate(w, p)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team")
	p := Page{Title: "services"}
	if t != nil {
		p.T = t.(Team)
	}
	renderTemplate(w, p)
}

func GetScores(w http.ResponseWriter, r *http.Request) {
	scores := DataGetAllScore()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		Logger.Println("Error encoding json:", err)
	}
}

func getChallenges() map[string]int {
	totals, err := DataGetTotalChallenges()
	if err != nil {
		Logger.Println("Could not get challenges:", err)
	}
	return totals
}

func getTeamChallenges(teamname string) map[string]int {
	acquired, err := DataGetTeamChallenges(teamname)
	if err != nil {
		Logger.Println("Could not get team challenges:", err)
	}
	return acquired
}

func getTeamScore(teamname string) int {
	return DataGetTeamScore(teamname)
}

func getAllTeamScores() []Result {
	return DataGetAllScore()
}

func renderTemplate(w http.ResponseWriter, p Page) {
	err := templates[p.Title].ExecuteTemplate(w, p.Title+".tmpl", &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
