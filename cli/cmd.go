package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

type sensor struct {
	ID          uuid.UUID `json:"id"`
	Activity    string    `json:"activity,omitempty"`
	Status      string    `json:"status,omitempty"`
	RAM         float64   `json:"ram,omitempty"`
	Ramusage    float64   `json:"ramusage,omitempty"`
	Loadaverage float64   `json:"loadaverage,omitempty"`
	Numcpu      float64   `json:"numcpu,omitempty"`
}

var (
	ids = []uuid.UUID{
		uuid.FromStringOrNil("1377959e-97ce-46c1-9715-22c34bb9afbe"),
		// uuid.FromStringOrNil("99f123d0-ad44-4752-9176-8f1ac547030c"),
		// uuid.FromStringOrNil("d8f722c7-8345-4396-a17a-7084c9af6745"),
		// uuid.FromStringOrNil("da60f79c-a1d5-4ce0-8bcc-ddc17860571f"),
		// uuid.FromStringOrNil("e8598b0d-a488-467b-9fe9-52026d65ada5"),
	}
	activities   = []string{"slow", "normal", "fast"}
	status       = []string{"running", "running", "running", "running", "offline", "failure"}
	rams         = []float64{2}
	ramusages    = []float64{0.3, 0.6, 1, 1.3, 1.6, 2}
	loardaerages = []float64{0.1, 0.2, 0.5, 0.8, 1, 1.1, 1.5, 2.2, 3.3, 4}
	numcpu       = []float64{4}
)

func genRandSensors() sensor {
	return sensor{
		ID: ids[rand.Intn(len(ids))],
		// Activity: activities[rand.Intn(len(activities))],
		Status: status[rand.Intn(len(status))],
		// RAM:         rams[rand.Intn(len(rams))],
		// Ramusage:    ramusages[rand.Intn(len(ramusages))],
		Loadaverage: loardaerages[rand.Intn(len(loardaerages))],
		Numcpu:      numcpu[0],
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage : %s <app_token> [https://host:port]\n", os.Args[0])
		os.Exit(1)
	}
	host := "http://localhost:2020"
	if len(os.Args) >= 3 {
		host = strings.TrimRight(os.Args[2], "/")
	}
	token := os.Args[1]

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}

	sch := make(chan sensor, 1)

	go func(ch chan<- sensor) {
		for {
			ch <- genRandSensors()
		}
	}(sch)

	for sensor := range sch {
		bstr, _ := json.Marshal(sensor)
		body := bytes.NewBuffer(bstr)

		req, err := http.NewRequest("POST", host+"/sensors", body)
		req.Header.Set("Content-type", "application/json")
		if err != nil {
			log.Println(err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)

		if err != nil {
			log.Printf("%s %v %s\n", resp.Status, err, bstr)
			continue
		}

		b, _ := ioutil.ReadAll(resp.Body)
		log.Printf("%s %s %s\n", resp.Status, b, bstr)
		resp.Body.Close()

		time.Sleep(time.Second)
	}
}
