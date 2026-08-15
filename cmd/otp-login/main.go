package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/infrai-examples/devtools-sms-otp/otpflow"
)

func main() {
	client, err := otpflow.NewSMSClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	pipeline := otpflow.NewLoginPipeline(client)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", handle(pipeline.Send))
	mux.HandleFunc("POST /login/verify", handle(pipeline.Verify))
	log.Printf("otp login service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handle(run func(context.Context, otpflow.LoginInput) (otpflow.LoginResult, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input otpflow.LoginInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := run(r.Context(), input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
