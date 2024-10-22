package main

import (
	"fmt"
	"flag"
	"bytes"
	"encoding/json"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"net/http"
	"log"
	"math/rand"
	"time"
)

type Patient struct {
	Port int
	PortsList []int
}

type Share struct {
	Share int
}

var client *http.Client
var hospitalPort int
var port int
var aggShare int
var maxCompVal int
var data int
var patients Patient
var maxRanVal int
var totPat int
var sharesReceived []int

func main() {
	flag.IntVar(&hospitalPort, "h", 8080, "port of the hospital")
	flag.IntVar(&port, "port", 8081, "port for patient")
	flag.IntVar(&maxCompVal, "maxCompVal", 300, "the computational max value")
	flag.IntVar(&totPat, "t", 3, "the total amount of patients")
	flag.Parse()

	maxRanVal = maxCompVal / 3

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data = r.Intn(maxRanVal)

	log.Println("Patient", port, "has data value", data) 

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

	// Start the patient server
	go patientServer()

	// Register patient with hospital
	log.Println("Patient", port, "registered with hospital")

	url := fmt.Sprintf("https://localhost:%d/patient", hospitalPort)

	patientInfo := Patient{
		Port: port,
	}

	b, err := json.Marshal(patientInfo)
	if err != nil {
		log.Fatal("Marshalling patient failed on port", port, "with error:", err)
	}

	response, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatal("Regisering with hospital failed on port ", port, " with error: ", err)
	}
	log.Println("Patient", port, "registered with hospital. Received response code", response.Status)

	select {} // Keep the server running
}

// server for patient
func patientServer() {
	log.Println("Creating patient server at port", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/patients", Patients)
	mux.HandleFunc("/shares", Shares)

	err := http.ListenAndServeTLS(FormatPort(port), "server.crt", "server.key", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func Patients(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		log.Println("Patient", port, "received POST from /patients")

		body, err := io.ReadAll(req.Body)
		if err != nil {
			handleError(w, port, err, "Reading request body failed during /patients")
			return
		}
		if err := json.Unmarshal(body, &patients); err != nil {
			handleError(w, port, err, "Unmarshalling patients failed")
			return
		}
		
		// n-out-of-n additive secret sharing
		shares := GenerateShares(maxRanVal, data, totPat) 

		log.Println("Patient", port, "received list of patients:", patients.PortsList)
		
		// Send shares to patients
		for index, shareValue := range shares {
			if index == totPat-1 {
				break
			}
			go func(index, shareValue int) {
				share := Share{
					Share: shareValue,
				}
				shareBytes, err := json.Marshal(share)
				if err != nil {
					handleError(w, port, err, "Marshalling share during /patients failed")
					return
				}
				url := fmt.Sprintf("https://localhost:%d/shares", patients.PortsList[index])
				response, err := client.Post(url, "application/json", bytes.NewReader(shareBytes))
				if err != nil {
					log.Println("Sending", port, "share failed to", patients.PortsList[index], ":", err)
					return
				}
				log.Println("Sent share to", patients.PortsList[index], "from", port, "Received response code:", response.StatusCode)
			}(index, shareValue)
		}
		// Append the last share to the receivedShares list
		sharesReceived = append(sharesReceived, shares[len(shares)-1])

		handleReceivedShare(w)
	}
}

func Shares(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		log.Println("Patient", port, "received POST from /shares")

		body, err := io.ReadAll(req.Body)
		if err != nil {
			handleError(w, port, err, "Reading request body from /shares failed")
			return
		}

		receivedShare := &Share{}
		err = json.Unmarshal(body, receivedShare)
		if err != nil {
			handleError(w, port, err, "Unmarshalling share failed")
			return
		}

		// Append the received share to the receivedShares list
		sharesReceived = append(sharesReceived, receivedShare.Share)

		handleReceivedShare(w)
	}
}

// sends the aggregate share to the hospital
func sendAggShare() {
	log.Println("Computing patient", port, "aggregate share")

	for _, share := range sharesReceived {
		aggShare = aggShare + share
	}

	log.Println("Patient", port, "aggregate share is", aggShare) //then send it to the hospital

	agg := Share{
		Share: aggShare,
	}

	b, err := json.Marshal(agg)
	if err != nil {
		log.Fatal("Marshalling", port, "aggregate share during /shares failed with error", err)
		return
	}

	log.Println("Patient", port, "sending aggregate share", aggShare, "to hospital")

	url := fmt.Sprintf("https://localhost:%d/shares", hospitalPort)

	response, err := client.Post(url, "string", bytes.NewReader(b))
	if err != nil {
		log.Fatal("Patient", port, "sending aggregate share to hospital failed with error", err)
		return
	}

	log.Println("Sent aggregate share to hospital from", port, "with response code:", response.StatusCode)
}

// GenerateShares with n-out-of-n additive secret share
func GenerateShares(p int, data int, amount int) []int {
	shares := make([]int, amount)
	totalShares := 0

	for i := 0; i < amount-1; i++ {
		share := rand.Intn(p) // random number between 0 and p
		shares[i] = share // assign the share to the shares array
		totalShares += share // add the share to the totalShares
	}

	// Calculate last share 
	shares[amount-1] = data - totalShares

	return shares
}

// helper function to format the port
func FormatPort(port int) string {
	return fmt.Sprintf(":%d", port)
}

// Helper function which checks if all shares have been received and sends the aggregate share to the hospital
func handleReceivedShare(w http.ResponseWriter) {
	// Check if all shares have been received
	if len(sharesReceived) == totPat {
		sendAggShare()
	}

	// Respond with status OK
	w.WriteHeader(http.StatusOK)
}

// Helper function to handle errors
func handleError(w http.ResponseWriter, port int, err error, message string) {
    http.Error(w, fmt.Sprintf("%d: %s: %v", port, message, err), http.StatusInternalServerError)
}
