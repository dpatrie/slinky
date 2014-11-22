package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var db gorm.DB

func main() {
	db = setupDb()

	db.DB().Ping()

	http.Handle("/", setupHTTP())

	bind := fmt.Sprintf("%s:%s", os.Getenv("HOST"), os.Getenv("PORT"))
	fmt.Printf("Listening on %s...\n\n", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Fatal(err)
	}
}

func setupDb() gorm.DB {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_USERNAME"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_PASSWORD"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_HOST"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_PORT"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB"),
	)
	log.Printf("Using %s", dsn)
	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %s", err)
	}

	db.AutoMigrate(&Url{})

	return db
}

func setupHTTP() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/urls", postUrls).Methods("POST").Headers("Content-Type", "application/json")
	r.HandleFunc("/urls/{key}", getUrls).Methods("GET")

	return handlers.CombinedLoggingHandler(os.Stdout, r)
}

func postUrls(w http.ResponseWriter, req *http.Request) {
	u := Url{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Server error", http.StatusBadRequest)
		log.Printf("ERROR: %s\n", err)
	}
	err = json.Unmarshal(body, &u)
	if err != nil {
		http.Error(w, "Server error", http.StatusBadRequest)
		log.Printf("ERROR: %s\n", err)
	}

	newURL := &Url{
		Href: u.Href,
	}

	db.Save(&newURL)

	out, err := json.Marshal(newURL)
	if err != nil {
		http.Error(w, "Server error", http.StatusBadRequest)
		log.Printf("ERROR: %s\n", err)
	}
	fmt.Fprintf(w, "%s", out)
}

func getUrls(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello GET")
}

//Url Model
type Url struct {
	Id        int64
	Href      string
	Key       string `sql:"size:32"`
	Vote      int64
	Score     int64
	AvgScore  float64 `sql:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

func (u *Url) BeforeCreate() error {
	u.Key = GetMD5Hash(fmt.Sprintf("%s-%d", u.Href, time.Now().Unix()))
	return nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
