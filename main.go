package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/ddymko/go-jsonerror"
	"github.com/joho/godotenv"
	"github.com/twilio/twilio-go/twiml"
)

func appError(w http.ResponseWriter, err error) {
	var error jsonerror.ErrorJSON
	error.AddError(jsonerror.ErrorComp{
		Detail: err.Error(),
		Code:   strconv.Itoa(http.StatusBadRequest),
		Title:  "Something went wrong",
		Status: http.StatusBadRequest,
	})
	http.Error(w, error.Error(), http.StatusBadRequest)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/xml")

	sender := r.FormValue("From")
	message := r.FormValue("Body")

	var twimlElements []twiml.Element
	if sender == os.Getenv("MY_PHONE_NUMBER") {
		var regex string = `^(?P<recipient>\+[1-9]\d{1,14}): (?P<message>.*)`
		re := regexp.MustCompile(regex)
		if re.MatchString(message) {
			result := make(map[string]string)
			match := re.FindStringSubmatch(message)
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			twiml := &twiml.MessagingMessage{
				Body: result["message"],
				To:   result["recipient"],
			}
			twimlElements = append(twimlElements, twiml)
		} else {
			twiml := &twiml.MessagingMessage{
				Body: "To reply to someone, you need to specify the recipient's phone number in E.164 format followed by a colon (':') and a space before the message, e.g.,: \"+16501231234: Here is my message to you.\".",
				To:   os.Getenv("MY_PHONE_NUMBER"),
			}
			twimlElements = append(twimlElements, twiml)
		}

		for _, value := range twimlElements {
			log.Printf("%+v\n", value)
		}
		response, err := twiml.Messages(twimlElements)
		if err != nil {
			appError(w, fmt.Errorf("could not prepare outgoing TwiML. reason: %s", err))
			return
		}

		w.Write([]byte(response))
		return
	}

	twimlElements = append(twimlElements, &twiml.MessagingMessage{
		Body: sender + ": " + message,
		To:   os.Getenv("MY_PHONE_NUMBER"),
	})
	response, err := twiml.Messages(twimlElements)
	if err != nil {
		appError(w, fmt.Errorf("could not prepare incoming TwiML. reason: %s", err))
		return
	}

	w.Write([]byte(response))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", handleRequest)

	log.Print("Starting server on :8080")
	err = http.ListenAndServe(":8080", mux)
	log.Fatal(err)
}
