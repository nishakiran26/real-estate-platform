package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var slugRe = regexp.MustCompile("[^a-z0-9]+")

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func GenerateOTP(length int) string {
	digits := "0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		result[i] = digits[n.Int64()]
	}
	return string(result)
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ParseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func Ptr(v interface{}) interface{} { return v }

func FormatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
