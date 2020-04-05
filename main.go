// https://gowebexamples.com/forms/
package main

import (
	"fmt"
	"github.com/kennygrant/sanitize"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"
	"viper"
)

type Contact struct {
	Name    string
	Email   string
	Message string
}

var limiter = NewIPRateLimiter(1, 1)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HandlePostRequest)
	log.Fatal(http.ListenAndServe(":8088", limitMiddleware(mux)))
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

func SendEmail(n string, e string, m string) {
	// https://app.sendgrid.com/guide/integrate
	from := mail.NewEmail("Digital Masterpiece", "noreply@digital-masterpiece.com")
	subject := "Contact Form Inquiry"
	to := mail.NewEmail(GetEnv("RECIPIENT_NAME"), GetEnv("RECIPIENT_EMAIL"))
	plainTextContent := fmt.Sprintf("Name: %s\r\n\r\nEmail: %s\r\n\r\nMessage: %s", n, e, m)
	htmlContent := fmt.Sprintf("Name: %s<br>Email: %s<br>Message: %s", n, e, m)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(GetEnv("SENDGRID_API_KEY"))

	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Headers)
	}
}

func HandlePostRequest(w http.ResponseWriter, r *http.Request) {
	// If the path isn't the root, 404.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", GetEnv("ALLOWED_ORIGIN"))
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	// Enforce timestamps in UTC.
	invalidRequestMessage := ": Invalid Request Received."
	now := GetUTCTime()

	fmt.Println(r.Header.Get("Origin"))

	if r.Header.Get("Origin") != GetEnv("ALLOWED_ORIGIN") {
		fmt.Println(now, invalidRequestMessage)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}

	// Enforce POST method.
	if r.Method != http.MethodPost {
		fmt.Println(now, invalidRequestMessage)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
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

	fmt.Println(now, ": Valid Request Received!")

	// Sanitize the inputs.
	name := sanitize.HTML(details.Name)
	email := sanitize.HTML(details.Email)
	message := sanitize.HTML(details.Message)

	SendEmail(name, email, message)
}

// https://dev.to/plutov/rate-limiting-http-requests-in-go-based-on-ip-address-542g
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

// AddIP creates a new rate limiter and adds it to the ip map using the IP address as the key.
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter
	return limiter
}

// GetLimiter returns the rate limiter for the provided IP address if it exists. Otherwise calls AddIP to add the IP address to the map.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]
	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}
	i.mu.Unlock()
	return limiter
}

func limitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := limiter.GetLimiter(r.RemoteAddr)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
