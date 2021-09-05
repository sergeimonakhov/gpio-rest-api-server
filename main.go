package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stianeikeland/go-rpio/v4"
)

type gpioRequest struct {
	Active bool `json:"active"`
}

type gpioStatusResponse struct {
	IsActive bool `json:"is_active"`
}

type gpio struct {
	GpioId   int  `json:"gpioID"`
	IsActive bool `json:"isActive"`
}

var (
	dbfile     string
	recovery   bool
	listenport int
	db         *sql.DB
)

func getGpioID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	pinId, err := strconv.Atoi(id)
	return pinId, err
}

func gpioPinEnable(pinId int, pinStatuses bool) {
	pin := rpio.Pin(pinId)
	pin.Output()

	if pinStatuses {
		pin.Low()
	} else {
		pin.High()
	}
}

func setGpioStatus(w http.ResponseWriter, r *http.Request) {
	var gr gpioRequest

	pinId, err := getGpioID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(reqBody, &gr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gpioPinEnable(pinId, gr.Active)
	insertGpioStatus(pinId, gr.Active)
}

func responseGpioStatus(w http.ResponseWriter, r *http.Request) {
	var gsr gpioStatusResponse

	pinId, err := getGpioID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gsr.IsActive = getGpioStatus(pinId)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(gsr)
}

func createInitTable() {
	createGpioTableSQL := `CREATE TABLE IF NOT EXISTS gpio (
		"gpioID" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"active" bool
		);`

	statement, err := db.Prepare(createGpioTableSQL)
	if err != nil {
		log.Println(err.Error())
	}
	defer statement.Close()

	statement.Exec()
}

func insertGpioStatus(gpioID int, active bool) {
	gpioSqlQuery := `INSERT INTO gpio(gpioID, active) VALUES (?, ?)
	                    ON CONFLICT(gpioID) DO UPDATE SET active=excluded.active;`
	statement, err := db.Prepare(gpioSqlQuery)
	if err != nil {
		log.Println(err.Error())
	}
	defer statement.Close()
	_, err = statement.Exec(gpioID, active)
	if err != nil {
		log.Println(err.Error())
	}
}

func getGpioStatus(gpioID int) bool {
	var active bool

	gpioSqlQuery := `SELECT active FROM gpio WHERE gpioID == ?`
	statement, err := db.Prepare(gpioSqlQuery)
	if err != nil {
		log.Println(err.Error())
	}
	defer statement.Close()
	err = statement.QueryRow(gpioID).Scan(&active)
	if err != nil {
		log.Println(err.Error())
		active = false
	}
	return active
}

func getGpioAllStatus() []gpio {
	gpios := []gpio{}
	rows, err := db.Query("SELECT gpioID, active FROM gpio")
	if err != nil {
		log.Println(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		g := gpio{}
		err := rows.Scan(&g.GpioId, &g.IsActive)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		gpios = append(gpios, g)
	}

	return gpios
}

func dbInit() {
	var err error

	if _, err = os.Stat(dbfile); os.IsNotExist(err) {
		file, err := os.Create(dbfile)
		if err != nil {
			log.Fatalln(err.Error())
		}
		file.Close()
	}

	db, err = sql.Open("sqlite3", dbfile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	createInitTable()
}

func gpioInit() {
	if err := rpio.Open(); err != nil {
		log.Fatalln(err)
	}
}
func recoveryGpioState() {
	gpiosStatus := getGpioAllStatus()
	for _, g := range gpiosStatus {
		gpioPinEnable(g.GpioId, g.IsActive)
	}
}

func init() {
	flag.StringVar(&dbfile, "dbfile", "./gpio.db", "Set db file")
	flag.BoolVar(&recovery, "recovery", false, "Recovery gpio state at start")
	flag.IntVar(&listenport, "listen-port", 8081, "Set http server listen port")
}

func main() {
	flag.Parse()

	//DB initialization
	dbInit()

	//GPIO init
	gpioInit()

	//Recovery gpio state
	if recovery {
		recoveryGpioState()
		log.Println("gpio state recovered")
	}

	//mux route
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/gpios/{id}", setGpioStatus).Methods("POST")
	myRouter.HandleFunc("/gpios/{id}", responseGpioStatus).Methods("GET")

	//Run http server
	log.Println("Starting HTTP server on port:", listenport)
	log.Println(http.ListenAndServe(":"+strconv.Itoa(listenport), myRouter))
}
