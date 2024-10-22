package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"log"
	"net/http"

)

type Patient struct {
	Port int
	PortsList []int
}

type Share struct {
	Share int
}

var client *http.Client
var patients []int
var totPat int
var regPat int
var data int
var port int
var sharesReceived int

func main() {
	flag.IntVar(&totPat, "t", 3, "total patients")
	flag.IntVar(&port, "port", 8080, "hospital port")
	flag.Parse()

	// cert load in
	cert, err := os.ReadFile("server.crt")
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	go hospitalServer()

	select {} // Keep the server running
}

// server for hospital
func hospitalServer() {
	log.Println("Creating hospital server at port", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/patient", Patients)
	mux.HandleFunc("/shares", Shares)

	err := http.ListenAndServeTLS(FormatPort(port), "server.crt", "server.key", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func Patients(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		log.Println("Hospital", port, "received POST from /patients")

		body, err := io.ReadAll(req.Body)
		if err != nil {
			handleError(w, port, err, "Reading request body from /patients failed")
			return
		}
		receivedPort := &Patient{}
		err = json.Unmarshal(body, receivedPort)
		if err != nil {
			handleError(w, port, err, "Unmarshalling port failed")
			return
		}
		log.Println("Hospital", port, "registered new patient at port", receivedPort.Port)
		patients = append(patients, receivedPort.Port)
		regPat++
		if regPat == totPat {
			log.Println("Hospital", port, "Sending ports to patients")
			for i, p := range patients {
				// Create a new slice with all the patients except the current one
				// and then send it to the current patient
				remainingPatients := append([]int{}, patients[:i]...)
				remainingPatients = append(remainingPatients, patients[i+1:]...)

				log.Println("Hospital", port, "sending ports", remainingPatients, "to", p)
				url := fmt.Sprintf("https://localhost:%d/patients", p)
				patientPorts := Patient{
					PortsList: remainingPatients,
				}

				b, err := json.Marshal(patientPorts)
				if err != nil {
					log.Fatal("Hospital", port, "marshalling patientPorts failed with error", err)
				}

				response, err := client.Post(url, "application/json", bytes.NewReader(b))
				if err != nil {
					log.Fatal("Hospital", port, "posting patientPorts failed to", p, "with error:", err)
				}
				log.Println("Hospital", port, "sent ports to ", p, "with response code", response.Status)
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func Shares(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		log.Println("Hospital", port, "received POST from /shares")

		body, err := io.ReadAll(req.Body)
		if err != nil {
			handleError(w, port, err, "Reading request body from /shares failed")
			return
		}
		share := &Share{}
		err = json.Unmarshal(body, share)
		if err != nil {
			handleError(w, port, err, "Unmarshalling share failed")
			return
		}

		data = data + share.Share
		sharesReceived++
		log.Println("Hospital", port, "received share", share.Share, "with a total of", sharesReceived)

		if sharesReceived == totPat {
			log.Println("Finished computing: Final value is", data)
		}
		w.WriteHeader(http.StatusOK)
	}
}

// helper function to format the port
func FormatPort(port int) string {
	return fmt.Sprintf(":%d", port)
}

// Helper function to handle errors
func handleError(w http.ResponseWriter, port int, err error, message string) {
    http.Error(w, fmt.Sprintf("%d: %s: %v", port, message, err), http.StatusInternalServerError)
}