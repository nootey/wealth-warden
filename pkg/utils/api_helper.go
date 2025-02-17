package utils

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Response struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

func SuccessMessage(message string, title string, code int) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Title:   title,
			Message: message,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, err = w.Write(jsonResponse)
		if err != nil {
			return
		}
	}
}

func ErrorMessage(title string, message string, code int) func(c *gin.Context, err error) {

	return func(c *gin.Context, err error) {
		response := struct {
			Title   string `json:"title"`
			Message string `json:"message"`
			Code    int    `json:"code"`
		}{
			Title:   title,
			Message: message,
			Code:    code,
		}
		if err != nil {
			prodMode := os.Getenv("RELEASE")
			prod, err2 := strconv.ParseBool(prodMode)
			if err2 != nil {
				prod = false
			}
			if prod == false {
				PrintError(err)
			}

		}
		c.JSON(code, response)
	}
}

// PrintError PrintRed prints the given text in red color.
func PrintError(err error) {
	fmt.Println("\033[31m" + err.Error() + "\033[0m")
}

func WarnError(message string) {
	fmt.Println("\033[31m" + message + "\033[0m")
}

// PrintSuccess PrintGreen prints the given text in green color.
func PrintSuccess(text string) {
	fmt.Println("\033[32m" + text + "\033[0m")
}

// PrintInfo PrintBlue prints the given text in blue color.
func PrintInfo(text string) {
	fmt.Println("\033[34;1m" + text + "\033[0m")
}

func FatalError(message string, err error) {
	log.Fatal("\033[31m"+message+"\033[0m", err)
}
