// https://gowebexamples.com/forms/
package main

import (
	"fmt"
	"github.com/kennygrant/sanitize"
	"log"
	"net/http"
	"regexp"
	"time"
)

type Contact struct {
	Name    string
	Email   string
	Message string
}

func validateParameter(r *regexp.Regexp, v string) bool {
	return v != "" && r.MatchString(v)
}

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	// Enforce timestamps in UTC.
	location, _ := time.LoadLocation("UTC")
	now := time.Now().In(location)

	// Enforce POST method.
	if r.Method != http.MethodPost {
		fmt.Println(now, ": Invalid Request Received.")
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// Bind the form values to the Contact struct.
	details := Contact{
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Message: r.FormValue("message"),
	}

	// Define validation regex.
	validNameRegex := regexp.MustCompile(`^.{2,}$`)
	validEmailRegex := regexp.MustCompile(`^.+@.+\..{2,}$`)

	// Validate the name and email address. Message is optional.
	if !validateParameter(validNameRegex, details.Name) {
		fmt.Println(now, ": Invalid Request Received.")
		http.Error(w, "Invalid name.", http.StatusBadRequest)
		return
	}

	if !validateParameter(validEmailRegex, details.Email) {
		fmt.Println(now, ": Invalid Request Received.")
		http.Error(w, "Invalid email address.", http.StatusBadRequest)
		return
	}

	// Sanitize the inputs.
	name := sanitize.HTML(details.Name)
	email := sanitize.HTML(details.Email)
	message := sanitize.HTML(details.Message)

	fmt.Println(name)
	fmt.Println(email)
	fmt.Println(message)

	fmt.Println(now, ": Valid Request Received!")
}

func main() {
	http.HandleFunc("/", handlePostRequest)
	log.Fatal(http.ListenAndServe(":8088", nil))
}
