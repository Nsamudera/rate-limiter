package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"rate_limiter/config"
	"rate_limiter/validator"
	"sync"
	"time"
)

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Do not change existing mocked data, as it might break the tests
var mockedRateLimiterConfig = map[string]validator.RateLimiterData{
	"PT A":    {Requests: 0, Limit: 3, Window: 5 * time.Second, FirstRequestTime: time.Now()},
	"PT B":    {Requests: 0, Limit: 3, Window: 3 * time.Second, FirstRequestTime: time.Now()},
	"PT TEST": {Requests: 0, Limit: 1, Window: 10 * time.Second, FirstRequestTime: time.Now()},
}

var rateLimiterData = validator.RateLimiterData{
	Requests:         config.DefaultRequest,
	Limit:            config.DefaultLimit,
	Window:           config.DefaultWindow,
	FirstRequestTime: time.Now(),
}
var rateLimiter = validator.RateLimiter{
	RateLimiterData: rateLimiterData,
	Mutex:           sync.Mutex{},
}

func main() {
	// Idea is to have a centralized place to view logs, which in this case is done via a file
	logFile, logErr := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logErr != nil {
		log.Fatal("Error in opening log file:", logErr)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/config", requestHandlerConfig)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error listening to port 8080:", err)
	}
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	currentTime := time.Now()
	clientID := r.Header.Get("clientID")
	response := Response{
		Status:  http.StatusOK,
		Message: fmt.Sprintf("Hello %v", clientID),
	}

	if !validator.ValidateClientID(clientID) {
		w.WriteHeader(http.StatusBadRequest)
		response.Status = http.StatusBadRequest
		response.Message = "No clientID provided"
		json.NewEncoder(w).Encode(response)
		return
	}
	rateLimiterCheck := rateLimiter.ValidateRequestLimit(clientID, currentTime, mockedRateLimiterConfig)

	if !rateLimiterCheck.Status {
		w.WriteHeader(http.StatusTooManyRequests)
		response.Status = http.StatusTooManyRequests
		response.Message = fmt.Sprintf("Too Many Requests for %v", clientID)
	} else {
		mockedRateLimiterConfig[clientID] = rateLimiterCheck.Data
	}
	json.NewEncoder(w).Encode(response)
}

func requestHandlerConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var response Response
	switch r.Method {
	// Current POST request will override any existing configuration
	// For future improvement, will need to add validation for existing clients and only allow editing of existing clients using PATCH request
	case "POST":
		clientID := r.Header.Get("clientID")
		var data validator.CreateData
		json.NewDecoder(r.Body).Decode(&data)

		if !validator.ValidateClientID(clientID) {
			w.WriteHeader(http.StatusBadRequest)
			response.Status = http.StatusBadRequest
			response.Message = "No clientID provided"
			json.NewEncoder(w).Encode(response)
			return
		}

		// For future improvement, be more specific in the error message
		// For example, if the body contains string, specify that the data type is incorrect
		if !validator.ValidateConfig(data) {
			w.WriteHeader(http.StatusBadRequest)
			response.Status = http.StatusBadRequest
			response.Message = "Config data must be greater than 0"
			json.NewEncoder(w).Encode(response)
			return
		}

		currentTime := time.Now()
		mockedRateLimiterConfig[clientID] = validator.RateLimiterData{
			Requests: 0, Limit: data.Limit, Window: time.Duration(data.Window) * time.Second, FirstRequestTime: currentTime,
		}

		response.Status = http.StatusOK
		response.Message = fmt.Sprintf("New config created for %v", clientID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		response.Status = http.StatusMethodNotAllowed
		response.Message = "Method not allowed"
	}

	json.NewEncoder(w).Encode(response)
}
