package commands

import (
	"encoding/xml"
	"github.com/dragosv/delta/xliff"
	guuid "github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

func openTestDatabase() {
	id := guuid.New()
	//
	testDatabase, err := openDatabase("sqlite3", "file:"+id.String()+"?mode=memory")
	if err != nil {
		panic("failed to connect database")
	}

	database = testDatabase
}

func createTestMapFs() {
	fs = afero.NewMemMapFs()
}

func setup() {
	createTestMapFs()
	openTestDatabase()

	source = path.Join("/delta/source", guuid.New().String())
	destination = path.Join("/delta/destination", guuid.New().String())
}

func TestRunPushCommand_NoFiles(t *testing.T) {
	setup()

	runPushCommand(source, destination)

	files := readDestinationDir()

	assert.Equal(t, 0, len(files))
}

func writeTestDocument(xliffTransUnit xliff.TransUnit) error {
	xliffFile := xliff.File{
		Original:       xliffTransUnit.Target.Language + ".xliff",
		SourceLanguage: xliffTransUnit.Source.Language,
		Datatype:       "plaintext",
		TargetLanguage: xliffTransUnit.Target.Language,
		Header:         xliff.Header{Tool: xliff.Tool{ToolID: "delta", ToolName: "delta", ToolVersion: "0.1", BuildNum: "0"}},
		Body:           xliff.Body{},
	}

	xliffDocument := xliff.Document{
		Version: "1.2",
	}

	xliffDocument.Files = append(xliffDocument.Files, xliffFile)

	xliffDocument.Files[0].Body.TransUnits = append(xliffDocument.Files[0].Body.TransUnits, xliffTransUnit)

	file, err := xml.MarshalIndent(xliffDocument, "", " ")

	if err != nil {
		return err
	}

	xliffPath := path.Join(source, xliffTransUnit.Target.Language+".xliff")

	err = afero.WriteFile(fs, xliffPath, file, 0644)

	return err
}

func readDocument(fileName string) (xliff.Document, error) {
	var data, error = afero.ReadFile(fs, fileName)

	if error != nil {
		return xliff.Document{}, error
	}

	return xliff.From(data)
}

var fileInfos map[string]os.FileInfo

func readDestinationDir() map[string]os.FileInfo {
	fileInfos = make(map[string]os.FileInfo)

	afero.Walk(fs, destination, readDirFunc)

	return fileInfos
}

func getPaths(fileInfos map[string]os.FileInfo) []string {
	var paths []string

	for path, _ := range fileInfos {
		paths = append(paths, path)
	}

	return paths
}

func readDirFunc(path string, info os.FileInfo, err error) error {
	if info != nil && !info.IsDir() {
		fileInfos[path] = info
	}

	return nil
}

func TestRunPushCommand_OneFile(t *testing.T) {
	setup()

	writeTestDocument(xliff.TransUnit{
		ID:      "679fc2df14fb48f39718a0c20392d259",
		Resname: "label.test",
		Source: xliff.Source{
			Data:     "test",
			Language: "en",
		},
		Target: xliff.Target{
			State:          "new",
			StateQualifier: "",
			Data:           "",
			Language:       "fr",
		},
		Notes: nil,
	})

	runPushCommand(source, destination)

	files := readDestinationDir()

	assert.Equal(t, 1, len(files))

	paths := getPaths(files)

	assert.Equal(t, "fr.xliff", files[paths[0]].Name())

	destinationDocument, error := readDocument(paths[0])

	assert.Nil(t, error)
	assert.Equal(t, 1, len(destinationDocument.Files))
	assert.Equal(t, 1, len(destinationDocument.Files[0].Body.TransUnits))

	destinationTransUnit := destinationDocument.Files[0].Body.TransUnits[0]

	assert.Equal(t, "label.test", destinationTransUnit.Resname)
	assert.Equal(t, "test", destinationTransUnit.Source.Data)
	assert.Equal(t, "en", destinationTransUnit.Source.Language)
	assert.Equal(t, destinationTransUnit.Target.State, "new")
	assert.Equal(t, destinationTransUnit.Target.Language, "fr")
}

func TestRunPushCommand_TranslatedFile(t *testing.T) {
	setup()

	writeTestDocument(xliff.TransUnit{
		ID:      "979fc2df14fb48f39718a0c20392d259",
		Resname: "label.test",
		Source: xliff.Source{
			Data:     "translated",
			Language: "en",
		},
		Target: xliff.Target{
			State:          "translated",
			StateQualifier: "",
			Data:           "traduit",
			Language:       "fr",
		},
		Notes: nil,
	})

	runPushCommand(source, destination)

	files := readDestinationDir()

	assert.Equal(t, 0, len(files))
}
