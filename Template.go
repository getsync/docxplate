package docxplate

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

// Template ..
type Template struct {
	path string
	zipr *zip.ReadCloser // zip reader

	// save all zip files here so we can build it again
	files map[string]*zip.File

	// only modified files (converted to []byte) save here
	modified map[string][]byte

	// hold all parsed params:values here
	params map[string]interface{}
}

// OpenTemplate ..
func OpenTemplate(docpath string) (*Template, error) {
	var err error

	t := &Template{
		path:     docpath,
		modified: map[string][]byte{},
	}

	// Unzip
	if t.zipr, err = zip.OpenReader(t.path); err != nil {
		return nil, err
	}

	// Get main document
	t.files = map[string]*zip.File{}
	for _, f := range t.zipr.File {
		t.files[f.Name] = f
	}
	if t.MainDocument() == nil {
		return nil, fmt.Errorf("mandatory [ word/document.xml ] not found")
	}

	return t, nil
}

// MainDocument ..
func (t *Template) MainDocument() *zip.File {
	fxml := t.files["word/document.xml"]
	return fxml
}

// ExportDocx - save new/modified docx based on template
func (t *Template) ExportDocx(path string) {
	fDocx, err := os.Create(path)
	if err != nil {
		return
	}
	defer fDocx.Close()

	zipw := zip.NewWriter(fDocx)
	defer zipw.Close()

	// Loop existing files to build docx archive again
	for _, f := range t.files {
		var err error

		// Read contents of single file inside zip
		var fr io.ReadCloser
		if fr, err = f.Open(); err != nil {
			log.Printf("Error reading [ %s ] from archive", f.Name)
			continue
		}
		fbuf := new(bytes.Buffer)
		fbuf.ReadFrom(fr)
		fr.Close()

		// Write contents as single file inside zip
		var fw io.Writer
		if fw, err = zipw.Create(f.Name); err != nil {
			log.Printf("Error writing [ %s ] to archive", f.Name)
			continue
		}

		// // Move/Write struct-saved file to docx archive file back
		// if f.Name == "word/document.xml" {
		//
		// 	// file to XML nodes struct
		// 	xmlNodes := t.fileToXMLStruct(f.Name)
		// 	buf := structToXMLBytes(xmlNodes)
		//
		// 	head := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
		// 	buf = append(head, buf...)
		// 	fw.Write(buf)
		//
		// 	ioutil.WriteFile("XXX.xml", buf, 0666) // DEBUG
		// 	continue
		// }

		// Move/Write struct-saved file to docx archive file back
		if buf, isModified := t.modified[f.Name]; isModified {

			// // file to XML nodes struct
			// xmlNodes := t.fileToXMLStruct(f.Name)
			// buf := structToXMLBytes(xmlNodes)

			// head := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
			// buf = append(head, buf...)
			fw.Write(buf)

			ioutil.WriteFile("XXX.xml", buf, 0666) // DEBUG
			continue
		}

		fw.Write(fbuf.Bytes())
	}

	return
}

// Convert given file (from template.Files) to struct of xml nodes
func (t *Template) fileToXMLStruct(fname string) *xmlDocument {
	f, ok := t.files[fname]
	if !ok {
		return nil
	}

	fr, _ := f.Open()
	buf := readerBytes(fr)

	// Do not strip <w: entiraly, but keep reference as w-t
	// So any string without w: would stay same, but all w- will be replaced again
	buf = bytes.Replace(buf, []byte("<w:"), []byte("<w-"), -1)
	buf = bytes.Replace(buf, []byte("</w:"), []byte("</w-"), -1)

	d := &xmlDocument{}
	err := xml.Unmarshal(buf, &d)
	if err != nil {
		color.Red("fileToXMLStruct: %v", err)
	}

	// color.Cyan("%s", structToXMLBytes(d))
	return d
}

// Params  - replace template placeholders with params
// "Hello {{ Name }}!"" --> "Hello World!""
func (t *Template) Params(v interface{}) {
	t.params = collectParams("", v)

	f := t.MainDocument()
	// doc := t.fileToXMLStruct(f.Name)

	// Params: slices
	for k, v := range t.params {
		vtype := fmt.Sprintf("%T", v)
		if vtype[:2] != "[]" {
			// skip non-slices
			continue
		}
		color.Magenta("%-20s %-10T %v", k, v, v)
	}

	// When replace massive simple params: single int, string or single Struct.string
	fr, _ := f.Open()
	buf := readerBytes(fr)

	for k, v := range t.params {
		pkey := fmt.Sprintf("{{%s}}", k)
		pval := fmt.Sprintf("%v", v)
		color.Green("%-20s %-10T %v", k, v, v)
		buf = bytes.Replace(buf, []byte(pkey), []byte(pval), -1)
	}
	t.modified[f.Name] = buf
}