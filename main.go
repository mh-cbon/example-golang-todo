package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	os.Remove("./foo.db")

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := NewController(NewTodoSlice())

	r := mux.NewRouter()

	s := r.PathPrefix("/todos").Subrouter()
	s.HandleFunc("/", server.todoIndex).Methods("GET")
	s.HandleFunc("/", server.todoCreate).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", server.todoShow).Methods("GET")
	s.HandleFunc("/{id:[0-9]+}", server.todoUpdate).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", server.todoDelete).Methods("DELETE")

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("static/"))))

	fmt.Println("Starting server on port 3000")
	http.ListenAndServe(":3000", r)
}

//go:generate lister gen_slice.go *Todo:*TodoSlice

// Todo "Object"
type Todo struct {
	ID          int    `json:"Id"`
	Title       string `json:"Title"`
	Category    string `json:"Category"`
	DtCreated   string `json:"Dt_created"`
	DtCompleted string `json:"Dt_completed"`
	State       string `json:"State"`
}

// GetID ...
func (t *Todo) GetID() int {
	return t.ID
}

// Controller store "context" values and connections in the server struct
type Controller struct {
	backend *TodoSlice
}

//go:generate channeler gen_channeler.go Controller:SyncController

//NewController is a ctor
func NewController(backend *TodoSlice) *Controller {
	return &Controller{backend: backend}
}

// Todo CRUD

func (s *Controller) todoIndex(res http.ResponseWriter, req *http.Request) {
	jsonResponse(res, s.backend.Get())
}

//
func (s *Controller) todoCreate(res http.ResponseWriter, req *http.Request) {
	todo := &Todo{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&todo)
	if err != nil {
		fmt.Println("ERROR decoding JSON - ", err)
		return
	}
	fmt.Println(todo)

	todo.ID = s.backend.Len()
	s.backend.Push(todo)

	jsonResponse(res, todo)
}

//
func (s *Controller) todoShow(res http.ResponseWriter, req *http.Request) {
	r, _ := regexp.Compile(`\d+$`)
	ID, err := strconv.Atoi(r.FindString(req.URL.Path))
	if !errorCheck(res, err) {
		todo := s.backend.Filter(FilterTodoSlice.ByID(ID)).First()
		fmt.Println(todo)
		jsonResponse(res, todo)
	}
}

//
func (s *Controller) todoUpdate(res http.ResponseWriter, req *http.Request) {
	todoParams := &Todo{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&todoParams)
	if err != nil {
		fmt.Println("ERROR decoding JSON - ", err)
		return
	}

	todoIndex := s.backend.Index(todoParams)
	if todoIndex == -1 {
		errorCheck(res, fmt.Errorf("Todo not found id=%v", todoParams.GetID()))
	} else {
		if !s.backend.RemoveAt(todoIndex) {
			errorCheck(res, fmt.Errorf("Todo not removed id=%v", todoParams.GetID()))
		}
		s.backend.InsertAt(todoIndex, todoParams)
	}

	jsonResponse(res, todoParams)
}

func (s *Controller) todoDelete(res http.ResponseWriter, req *http.Request) {
	r, _ := regexp.Compile(`\d+$`)
	ID, err := strconv.Atoi(r.FindString(req.URL.Path))
	if !errorCheck(res, err) {
		todoParams := &Todo{ID: ID}
		todoIndex := s.backend.Index(todoParams)
		if todoIndex == -1 {
			errorCheck(res, fmt.Errorf("Todo not found id=%v", todoParams.GetID()))
		} else {
			if !s.backend.RemoveAt(todoIndex) {
				errorCheck(res, fmt.Errorf("Todo not removed id=%v", todoParams.GetID()))
			}
		}
	}
	res.WriteHeader(200)
}

func jsonResponse(res http.ResponseWriter, data interface{}) {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")

	payload, err := json.Marshal(data)
	if errorCheck(res, err) {
		return
	}

	fmt.Fprintf(res, string(payload))
}

func errorCheck(res http.ResponseWriter, err error) bool {
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}
