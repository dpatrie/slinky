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
	serveHTTP()
}

func setupDb() gorm.DB {
	// dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?connect_timeout=2&sslmode=disable",
	// 	os.Getenv("OPENSHIFT_POSTGRESQL_DB_USERNAME"),
	// 	os.Getenv("OPENSHIFT_POSTGRESQL_DB_PASSWORD"),
	// 	os.Getenv("OPENSHIFT_POSTGRESQL_DB_HOST"),
	// 	os.Getenv("OPENSHIFT_POSTGRESQL_DB_PORT"),
	// 	os.Getenv("OPENSHIFT_POSTGRESQL_DB"),
	// )

	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s connect_timeout=2 sslmode=disable",
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_USERNAME"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_PASSWORD"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_HOST"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB_PORT"),
		os.Getenv("OPENSHIFT_POSTGRESQL_DB"),
	)
	// log.Printf("Using %s", dsn)
	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %s", err)
	}

	db.AutoMigrate(&Url{})

	return db
}

func serveHTTP() {
	r := mux.NewRouter()

	r.HandleFunc("/urls", postUrls).Methods("POST").Headers("Content-Type", "application/json")
	r.HandleFunc("/urls/{key}", getUrls).Methods("GET")

	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	bind := fmt.Sprintf("%s:%s", os.Getenv("HOST"), os.Getenv("PORT"))
	fmt.Printf("Listening on %s...\n\n", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Fatal(err)
	}
}

func postUrls(w http.ResponseWriter, req *http.Request) {
	u := Url{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &u)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	newURL := Url{
		Href: u.Href,
	}

	if err = db.Save(&newURL).Error; err != nil {
		log.Printf("ERROR: %s\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	log.Printf("/urls/%s", newURL.Key)
	http.Redirect(w, req, fmt.Sprintf("/urls/%s", newURL.Key), http.StatusSeeOther)
}

func getUrls(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	u := Url{}

	q := db.Where("key = ?", vars["key"]).First(&u)
	if q.RecordNotFound() {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	} else if q.Error != nil {
		log.Printf("ERROR: %s\n", q.Error)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	out, _ := json.Marshal(u)
	fmt.Fprintf(w, "%s", out)
}

//Url Model
type Url struct {
	Id        int64  `json:"-"`
	Key       string `sql:"size:32;not null;unique"`
	Href      string `sql:"not null;unique"`
	View      int64
	Score     int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time `json:"-"`
}

func (u *Url) BeforeCreate() error {
	u.Key = GetMD5Hash(u.Href)
	return nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
