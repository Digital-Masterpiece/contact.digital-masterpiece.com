// https://gowebexamples.com/forms/
package main

import (
	"fmt"
	"github.com/kennygrant/sanitize"
	"log"
	"net/http"
	"regexp"
	"time"
	"viper"
)

type Contact struct {
	Name    string
	Email   string
	Message string
}

func GetEnv(k string) string {
	viper.SetConfigFile(".env")
	e := viper.ReadInConfig()

	if e != nil {
		log.Fatalf("Error reading configuration file %s", e)
	}

	v, s := viper.Get(k).(string)

	if !s {
		log.Fatalf("Invalid type assertion.")
	}

	return v
}

func GetUTCTime() time.Time {
	location, _ := time.LoadLocation("UTC")
	return time.Now().In(location)
}


func ValidateParameter(r *regexp.Regexp, v string) bool {
	return v != "" && r.MatchString(v)
}

func HandlePostRequest(w http.ResponseWriter, r *http.Request) {
	// Enforce timestamps in UTC.
	invalidRequestMessage := ": Invalid Request Received."
	now := GetUTCTime()

	// Enforce POST method.
	if r.Method != http.MethodPost {
		fmt.Println(now, invalidRequestMessage)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// Bind the form values to a Contact struct.
	details := Contact{
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Message: r.FormValue("message"),
	}

	// Define validation regex.
	validNameRegex := regexp.MustCompile(`^.{2,}$`)
	validEmailRegex := regexp.MustCompile(`^.+@.+\..{2,}$`)

	// Validate the name and email address. Message is optional.
	if !ValidateParameter(validNameRegex, details.Name) {
		fmt.Println(now, invalidRequestMessage)
		http.Error(w, "Invalid name.", http.StatusBadRequest)
		return
	}

	if !ValidateParameter(validEmailRegex, details.Email) {
		fmt.Println(now, invalidRequestMessage)
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
	http.HandleFunc("/", HandlePostRequest)
	log.Fatal(http.ListenAndServe(":8088", nil))
}
