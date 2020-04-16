// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package xliff

import (
	"encoding/xml"
	"fmt"
)

type Tool struct {
	ToolID      string `xml:"tool-id,attr"`
	ToolName    string `xml:"tool-name,attr"`
	ToolVersion string `xml:"tool-version,attr"`
	BuildNum    string `xml:"build-num,attr"`
}

type Header struct {
	Tool Tool `xml:"tool"`
}

type TransUnit struct {
	ID      string `xml:"id,attr"`
	Resname string `xml:"resname,attr"`
	Source  Source `xml:"source"`
	Target  Target `xml:"target"`
	Note    string `xml:"note"`
}

type Source struct {
	Data     string `xml:",chardata"`
	Language string `xml:"lang,attr"`
}

type Target struct {
	State          string `xml:"state,attr"`
	StateQualifier string `xml:"state-qualifier,attr"`
	Data           string `xml:",chardata"`
	Language       string `xml:"lang,attr"`
}

type Body struct {
	TransUnits []TransUnit `xml:"trans-unit"`
}

type File struct {
	Original       string `xml:"original,attr"`
	SourceLanguage string `xml:"source-language,attr"`
	Datatype       string `xml:"datatype,attr"`
	TargetLanguage string `xml:"target-language,attr"`
	Header         Header `xml:"header"`
	Body           Body   `xml:"body"`
}

type Document struct {
	Version string `xml:"version,attr"`
	Files   []File `xml:"file"`
}

type ValidationErrorCode int

const (
	UnsupportedVersion ValidationErrorCode = iota
	MissingOriginalAttribute
	MissingSourceLanguage
	MissingTargetLanguage
	UnsupportedDatatype
	InconsistentSourceLanguage
	InconsistentTargetLanguage
	MissingTransUnitID
	MissingTransUnitSource
	MissingTransUnitTarget
)

type ValidationError struct {
	Code    ValidationErrorCode
	Message string
}

func (ve ValidationError) Error() string {
	code := "Unknown"
	switch ve.Code {
	case UnsupportedVersion:
		code = "UnsupportedVersion"
	case MissingOriginalAttribute:
		code = "MissingOriginalAttribute"
	case MissingSourceLanguage:
		code = "MissingSourceLanguage"
	case MissingTargetLanguage:
		code = "MissingTargetLanguage"
	case UnsupportedDatatype:
		code = "UnsupportedDatatype"
	case InconsistentSourceLanguage:
		code = "InconsistentSourceLanguage"
	case InconsistentTargetLanguage:
		code = "InconsistentTargetLanguage"
	case MissingTransUnitID:
		code = "MissingTransUnitID"
	case MissingTransUnitSource:
		code = "MissingTransUnitSource"
	case MissingTransUnitTarget:
		code = "MissingTransUnitTarget"
	}
	return fmt.Sprintf("%s: %s", code, ve.Message)
}

func From(data []byte) (Document, error) {
	var document Document
	if err := xml.Unmarshal(data, &document); err != nil {
		return Document{}, err
	}

	return document, nil
}

// Returns true if the document passes some basic consistency checks.
func (d Document) Validate() []ValidationError {
	var errors []ValidationError

	// Make sure the document is a version we understand
	if d.Version != "1.2" {
		errors = append(errors, ValidationError{
			Code:    UnsupportedVersion,
			Message: fmt.Sprintf("Version %s is not supported", d.Version),
		})
	}

	// Make sure all files have the attributes we need
	for idx, file := range d.Files {
		if file.Original == "" {
			errors = append(errors, ValidationError{
				Code:    MissingOriginalAttribute,
				Message: fmt.Sprintf("File #%d is missing 'original' attribute", idx),
			})
		}
		if file.SourceLanguage == "" {
			errors = append(errors, ValidationError{
				Code:    MissingSourceLanguage,
				Message: fmt.Sprintf("File '%s' is missing 'source-language' attribute", file.Original),
			})
		}
		if file.TargetLanguage == "" {
			errors = append(errors, ValidationError{
				Code:    MissingTargetLanguage,
				Message: fmt.Sprintf("File '%s' is missing 'target-language' attribute", file.Original),
			})
		}
		if file.Datatype != "plaintext" {
			errors = append(errors, ValidationError{
				Code: UnsupportedDatatype,
				Message: fmt.Sprintf("File '%s' has unsupported 'datatype' attribute with value '%s'",
					file.Original, file.Datatype),
			})
		}
	}

	// Make sure all files are consistent with source and target language
	sourceLanguage, targetLanguage := d.Files[0].SourceLanguage, d.Files[0].TargetLanguage
	for _, file := range d.Files {
		if file.SourceLanguage != sourceLanguage {
			errors = append(errors, ValidationError{
				Code: InconsistentSourceLanguage,
				Message: fmt.Sprintf("File '%s' has inconsistent 'source-language' attribute '%s'",
					file.Original, file.SourceLanguage),
			})
		}
		if file.TargetLanguage != targetLanguage {
			errors = append(errors, ValidationError{
				Code: InconsistentTargetLanguage,
				Message: fmt.Sprintf("File '%s' has inconsistent 'target-language' attribute '%s'",
					file.Original, file.TargetLanguage),
			})
		}
	}

	// Make sure all trans units have the attributes and children we expect
	for _, file := range d.Files {
		for idx, transUnit := range file.Body.TransUnits {
			if transUnit.ID == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitID,
					Message: fmt.Sprintf("Translation unit #%d in file '%s' is missing 'id' attribute",
						idx, file.Original),
				})
			}
			if transUnit.Source.Data == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitSource,
					Message: fmt.Sprintf("Translation unit '%s' in file '%s' is missing 'source' attribute",
						transUnit.ID, file.Original),
				})
			}
			if transUnit.Target.Data == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitTarget,
					Message: fmt.Sprintf("Translation unit '%s' in file '%s' is missing 'target' attribute",
						transUnit.ID, file.Original),
				})
			}
			if transUnit.Target.Language != file.TargetLanguage {
				errors = append(errors, ValidationError{
					Code: InconsistentTargetLanguage,
					Message: fmt.Sprintf("Translation unit '%s' in file '%s' is has a different target language attribute",
						transUnit.ID, file.Original),
				})
			}
		}
	}

	return errors
}

// Returns true if all translation units in all files have both a
// non-empty source and target.
func (d Document) IsComplete() bool {
	for _, file := range d.Files {
		for _, transUnit := range file.Body.TransUnits {
			if transUnit.Source.Data == "" || transUnit.Target.Data == "" || (transUnit.Target.State != "translated" && transUnit.Target.State != "signed-off") {
				return false
			}
		}
	}
	return true
}

// Returns true if all translation units in all files have both a
// non-empty source and target.
func (d Document) IncompleteTransUnits() []TransUnit {
	var transUnits []TransUnit

	for _, file := range d.Files {
		for _, transUnit := range file.Body.TransUnits {
			if transUnit.Source.Data == "" || transUnit.Target.Data == "" || (transUnit.Target.State != "translated" && transUnit.Target.State != "signed-off") {
				transUnits = append(transUnits, transUnit)
			}
		}
	}

	return transUnits
}

func (d Document) File(original string) (File, bool) {
	for _, file := range d.Files {
		if file.Original == original {
			return file, true
		}
	}
	return File{}, false
}
