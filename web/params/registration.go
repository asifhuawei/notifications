package params

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"

	"github.com/cloudfoundry-incubator/notifications/models"
)

var kindIDFormat = regexp.MustCompile(`^[0-9a-zA-Z_\-.]+$`)

type Registration struct {
	SourceDescription string        `json:"source_description"`
	Kinds             []models.Kind `json:"kinds"`
	IncludesKinds     bool
}

func NewRegistration(body io.Reader) (Registration, error) {
	var registration Registration
	var hashParams map[string]interface{}

	hashReader := bytes.NewBuffer([]byte{})
	structReader := bytes.NewBuffer([]byte{})
	io.Copy(io.MultiWriter(hashReader, structReader), body)

	err := json.NewDecoder(hashReader).Decode(&hashParams)
	if err != nil {
		return registration, ParseError{}
	}

	err = json.NewDecoder(structReader).Decode(&registration)
	if err != nil {
		return registration, ParseError{}
	}

	if _, ok := hashParams["kinds"]; ok {
		registration.IncludesKinds = true
	}

	return registration, nil
}

func (registration Registration) Validate() error {
	errors := ValidationError{}
	if registration.SourceDescription == "" {
		errors = append(errors, `"source_description" is a required field`)
	}

	kindErrors := ValidationError{}
	for _, kind := range registration.Kinds {
		if kind.ID == "" {
			kindErrors = append(kindErrors, `"kind.id" is a required field`)
		} else if !kindIDFormat.MatchString(kind.ID) {
			kindErrors = append(kindErrors, `"kind.id" is improperly formatted`)
		}

		if kind.Description == "" {
			kindErrors = append(kindErrors, `"kind.description" is a required field`)
		}

		if len(kindErrors) > 0 {
			break
		}
	}

	errors = append(errors, kindErrors...)

	if len(errors) > 0 {
		return errors
	}
	return nil
}
