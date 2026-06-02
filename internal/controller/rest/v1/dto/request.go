package dto

import (
	"encoding/json"
	"net/http"
)

// BindJSON decodes the JSON body of a request into target.
func BindJSON(r *http.Request, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}
