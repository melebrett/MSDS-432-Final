package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type TaxiTrips struct {
	TripID             string
	TaxiID             string
	TripStartTimestamp time.Time
	PickupLatitude     float64
	PickupLongitude    float64
	DropoffLatitude    float64
	DropoffLongitude   float64
	PickupLocationName string
	DropoffZipCode     int
}

type Nominatim struct {
	PlaceId     int              `json:"place_id"`
	Category    string           `json:"category"`
	DisplayName string           `json:"display_name"`
	Address     NominatimAddress `json:"address"`
	Boundingbox []string         `json:"boundingbox"`
}

type NominatimAddress struct {
	HomeNumber    int    `json:"house_number"`
	Road          string `json:"road"`
	Neighbourhood string `json:"neighbourhood"`
	Suburb        string `json:"suburb"`
	City          string `json:"city"`
	Municipality  string `json:"municipality"`
	County        string `json:"county"`
	Postcode      string `json:"postcode"`
}

var Trips []TaxiTrips
var AirportTrips []TaxiTrips

func DLConnect() (*sql.DB, error) {
	//Retreiving DB connection credential environment variables
	err := godotenv.Load(".env")
	var DLHOST = os.Getenv("DLHOST")
	var DLPORT = os.Getenv("DLPORT")
	var DLUSER = os.Getenv("DLUSER")
	var DLPASSWORD = os.Getenv("DLPASSWORD")
	var DLDBNAME = os.Getenv("DLDBNAME")
	if err != nil {
		log.Println("Could not load .env file")
	}

	DB_DSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", DLHOST, DLPORT, DLUSER, DLPASSWORD, DLDBNAME)

	db, err := sql.Open("postgres", DB_DSN)

	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Data Lake")

	return db, nil
}

func DMConnect() (*sql.DB, error) {
	//Retreiving DB connection credential environment variables
	err := godotenv.Load(".env")
	var DMHOST = os.Getenv("DMHOST")
	var DMPORT = os.Getenv("DMPORT")
	var DMUSER = os.Getenv("DMUSER")
	var DMPASSWORD = os.Getenv("DMPASSWORD")
	var DMDBNAME = os.Getenv("DMDBNAME")
	if err != nil {
		log.Println("Could not load .env file")
	}

	DB_DSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", DMHOST, DMPORT, DMUSER, DMPASSWORD, DMDBNAME)

	db, err := sql.Open("postgres", DB_DSN)

	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Data Mart")

	return db, nil
}

func String2Float(s string) float64 {
	value, _ := strconv.ParseFloat(s, 64)
	return value
}

func String2Timestamp(s string) time.Time {
	// '2023-01-01T00:00:00.000'
	const format = "2006-01-02T15:04:05.000"
	timestamp, err := time.Parse(format, s)
	if err != nil {
		log.Println("Error converting timestamp: ", timestamp, err)
	}
	return timestamp
}

func query_taxis() []TaxiTrips {
	db, err := DLConnect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	statement := `SELECT TripID, TaxiID, TripStartTimestamp, PickupCentroidLatitude, PickupCentroidLongitude, DropoffCentroidLatitude, DropoffCentroidLongitude FROM taxi_trips`

	rows, err := db.Query(statement)
	if err != nil {
		log.Fatal("Error querying database: ", err)
	}

	Data := []TaxiTrips{}

	for rows.Next() {
		var tripID string
		var taxiID string
		var startTimestamp string
		var pickupLatitude string
		var pickupLongitude string
		var dropoffLatitude string
		var dropoffLongitude string
		err = rows.Scan(&tripID, &taxiID, &startTimestamp, &pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude)
		if err != nil {
			log.Fatal("Scan error", err)
		}
		temp := TaxiTrips{TripID: tripID, TaxiID: taxiID, TripStartTimestamp: String2Timestamp(startTimestamp), PickupLatitude: String2Float(pickupLatitude), PickupLongitude: String2Float(pickupLongitude), DropoffLatitude: String2Float(dropoffLatitude), DropoffLongitude: String2Float(dropoffLongitude)}

		Data = append(Data, temp)
	}

	defer rows.Close()

	return Data
}

func GetLocationName(userAgent string, lat, lon float64) string {
	var myresults Nominatim
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=jsonv2&lat=%f&lon=%f", lat, lon)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println(resp.StatusCode)
	}

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	json.Unmarshal(resBody, &myresults)

	return myresults.DisplayName
}

func GetZipCode(userAgent string, lat, lon float64) string {
	var myresults Nominatim
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=jsonv2&lat=%f&lon=%f", lat, lon)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {

		fmt.Println(resp.StatusCode)

	}

	resBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {

		fmt.Println(err)

	}

	json.Unmarshal(resBody, &myresults)

	return myresults.Address.Postcode
}

func CreateDataMartTable() {
	db, err := DMConnect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	dropTableStatement := "DROP TABLE IF EXISTS requirement_2_airport_trips;"

	_, err = db.Exec(dropTableStatement)
	if err != nil {
		panic(err)
	}

	createTableStatement := `CREATE TABLE requirement_2_airport_trips (
								TripID               TEXT PRIMARY KEY,
								TaxiID               TEXT,
								TripStartTimestamp   TIMESTAMPTZ,
								PickupLatitude       FLOAT,
								PickupLongitude      FLOAT,
								PickupLocationName	 TEXT,
								DropoffLatitude      FLOAT,
								DropoffLongitude     FLOAT,
								DropoffZipCode       INTEGER
							);`

	_, err = db.Exec(createTableStatement)
	if err != nil {
		panic(err)
	}
}

func LoadToDataMart(tripRecords []TaxiTrips) {
	db, err := DMConnect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	insertStatement := `INSERT INTO requirement_2_airport_trips (TripID, TaxiID, TripStartTimestamp, PickupLatitude, PickupLongitude, PickupLocationName, DropoffLatitude,	DropoffLongitude, DropoffZipCode) 
							values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
							ON CONFLICT (TripID) 
							DO NOTHING;`

	for _, v := range tripRecords {
		_, err = db.Exec(insertStatement, v.TripID, v.TaxiID, v.TripStartTimestamp, v.PickupLatitude, v.PickupLongitude, v.PickupLocationName, v.DropoffLatitude, v.DropoffLongitude, v.DropoffZipCode)
		if err != nil {
			log.Println("Error inserting record, TripID = ", v.TripID, err)
		}
	}
}

func TestInsertion() {
	db, err := DMConnect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	testStatement1 := "SELECT DropoffZipCode FROM requirement_2_airport_trips LIMIT 10"
	rows, err := db.Query(testStatement1)
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var testzipcode string
		err = rows.Scan(&testzipcode)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(testzipcode)
	}
}

func main() {
	Trips = query_taxis()

	// Reverse geocode to return pickup location name. If location name matches one of the airports, add those records to new slice
	for _, v := range Trips {
		LocationName := GetLocationName("msds432-final-group-4", v.PickupLatitude, v.PickupLongitude)

		if LocationName == "O'Hare International Airport, 10000, Perimeter Road, O'Hare, Chicago, Jefferson Township, Cook County, Illinois, 60666, United States" || LocationName == "Lot A, O'Hare Commercial Departure, O'Hare, Chicago, Jefferson Township, Cook County, Illinois, 60666, United States" || LocationName == "Chicago Midway International Airport, 5700, South Cicero Avenue, Chicago, Illinois, 60638, United States" {
			v.PickupLocationName = LocationName
			AirportTrips = append(AirportTrips, v)
		}
	}

	// For trips from airport, reverse geocode to get dropoff zip code and update struct with zipcode field
	for i := 0; i < len(AirportTrips); i++ {
		record := &AirportTrips[i]
		zip, err := strconv.Atoi(GetZipCode("msds432-final-group-4", record.DropoffLatitude, record.DropoffLongitude))
		if err != nil {
			log.Println("Error converting zip to integer: ", err)
		}
		record.DropoffZipCode = zip
	}

	// Insert to Data Mart
	CreateDataMartTable()

	LoadToDataMart(AirportTrips)

	// // Testing successful ingestion to Data Mart
	// TestInsertion()
}
