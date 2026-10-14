package main

import (
	"log"
	"net/http"
)

func main() {
	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux(newOTPStore())))
}

func newMux(store *otpStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /send-otp", sendOTPHandler(store))
	mux.HandleFunc("POST /verify-otp", verifyOTPHandler(store))
	mux.HandleFunc("POST /login", loginHandler(store))
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
