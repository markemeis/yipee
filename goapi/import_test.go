package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type ImportPayload struct {
	ImportFile string `json:"importFile"`
	Name       string `json:"name"`
}

func makeImportRequest(
	t *testing.T,
	fname, impname string,
	save bool) *http.Request {

	fbytes := readTestData(fname)
	payload := &ImportPayload{base64.StdEncoding.EncodeToString(fbytes), impname}
	plbytes := toJsonBytes(payload)
	impurl := "/import"
	if save {
		impurl += "?save=true"
	}
	return httptest.NewRequest(http.MethodPost, impurl, bytes.NewReader(plbytes))
}

func assertImportSuccess(t *testing.T, result *ObjResp, expectname string) {
	flatFile := result.Data[0]["flatFile"].(map[string]interface{})
	ai := flatFile["app-info"].([]interface{})
	appinf := ai[0].(map[string]interface{})
	gotname := appinf["name"]
	if expectname != gotname {
		t.Errorf("appname mismatch: expected '%s', got '%s'", expectname, gotname)
	}
	if containers, ok := flatFile["container"].([]interface{}); ok {
		if len(containers) < 1 {
			t.Errorf("no containers for %s", expectname)
		}
	} else {
		t.Errorf("can't get containers for %s", expectname)
	}
}

type importTest struct {
	appname  string
	filename string
}

func TestImports(t *testing.T) {
	successTests := []*importTest{
		&importTest{"bday-from-yaml", "bday4.yml"},
		&importTest{"testbday-tar", "bday.tgz"},
		// &importTest{"from-compose", "compose.yml"},
	}
	for _, td := range successTests {
		req := makeImportRequest(t, td.filename, td.appname, false)
		result := doSuccessRequest(t, req)
		assertImportSuccess(t, result, td.appname)
	}

	failTests := []*importTest{
		&importTest{"badbday-from-yaml", "badbday.yml"},
		&importTest{"badbday-tar", "badbday.tgz"},
		&importTest{"bad-from-compose", "badcompose.yml"},
	}
	for _, td := range failTests {
		req := makeImportRequest(t, td.filename, td.appname, false)
		eresult := doErrRequest(t, req)
		emsg := eresult.Data[0]
		if strings.Index(emsg, "is not defined in the schema") == -1 {
			t.Error("missing expected schema validation error", emsg)
		}
	}
}

func TestImportCache(t *testing.T) {
	// assert multiple import/save ops are allowed
	var guids []string
	for i := 0; i < 5; i++ {
		appname := fmt.Sprintf("cached-import-test-%d", i)
		req := makeImportRequest(t, "bday4.yml", appname, true)
		result := doSuccessRequest(t, req)
		guid := result.Data[0]["guid"].(string)
		guids = append(guids, guid)
	}
	for i, guid := range guids {
		// retrieve flatfile for guid
		req := httptest.NewRequest(http.MethodGet, "/import/"+guid, nil)
		result := doSuccessRequest(t, req)
		assertImportSuccess(t, result, fmt.Sprintf("cached-import-test-%d", i))

		// another GET of the same guid should fail
		eresult := doErrRequest(t, req)
		emsg := eresult.Data[0]
		expectMsg := fmt.Sprintf("%s: %s", CACHE_FETCH_ERR, guid)
		if expectMsg != emsg {
			t.Errorf("cache fetch error expected '%s', got '%s'", expectMsg, emsg)
		}
	}
}

func TestImportCacheBoundaries(t *testing.T) {
	importCache = NewCache(1, 2) // XXX

	// do an import/save
	req := makeImportRequest(t, "bday4.yml", "cached-import", true)
	result := doSuccessRequest(t, req)
	guid, ok := result.Data[0]["guid"].(string)
	if !ok {
		t.Error("no guid for import/save")
	}

	// second one should fail since we created a tiny cache here
	req = makeImportRequest(t, "bday4.yml", "cached-import", true)
	eresult := doErrRequest(t, req)
	emsg := eresult.Data[0]
	if CACHE_LIMIT_ERR != emsg {
		t.Errorf("cache limit error expected '%s', got '%s'", CACHE_LIMIT_ERR, emsg)
	}

	// assert cached value was removed after timeout, even if it was
	// never retrieved
	time.Sleep(3 * time.Second)
	req = httptest.NewRequest(http.MethodGet, "/import/"+guid, nil)
	eresult = doErrRequest(t, req)
	emsg = eresult.Data[0]
	expectMsg := fmt.Sprintf("%s: %s", CACHE_FETCH_ERR, guid)
	if expectMsg != emsg {
		t.Errorf("cache fetch error expected '%s', got '%s'", expectMsg, emsg)
	}
}

type helmImportTest struct {
	appname    string
	filename   string
	valuesname string
}

type helmImportFailTest struct {
	appname    string
	filename   string
	valuesname string
	errmsg     string
}

type HelmImportPayload struct {
	Chart          string `json:"chart"`
	Name           string `json:"name"`
	ValuesFilename string `json:"valuesFilename"`
}

func makeImportHelmRequest(
	t *testing.T,
	fname, impname, vname string,
	save bool) *http.Request {
	fbytes := readTestData(fname)
	payload := &HelmImportPayload{base64.StdEncoding.EncodeToString(fbytes),
		impname, vname}
	plbytes := toJsonBytes(payload)
	impurl := "/importhelm"
	if save {
		impurl += "?save=true"
	}
	return httptest.NewRequest(http.MethodPost, impurl, bytes.NewReader(plbytes))
}

func runImportHelmSuccessTests(t *testing.T, tests []*helmImportTest) {
	for _, td := range tests {
		fmt.Printf("Importing: %s\n", td.appname)
		// no cache
		req := makeImportHelmRequest(t, td.filename, td.appname, td.valuesname,
			false)
		result := doSuccessRequest(t, req)
		assertImportSuccess(t, result, td.appname)
		// cache
		req = makeImportHelmRequest(t, td.filename, td.appname, td.valuesname,
			true)
		result = doSuccessRequest(t, req)
		guid := result.Data[0]["guid"].(string)
		req = httptest.NewRequest(http.MethodGet, "/import/"+guid, nil)
		result = doSuccessRequest(t, req)
		assertImportSuccess(t, result, td.appname)
		// one shot - fail next time
		eresult := doErrRequest(t, req)
		emsg := eresult.Data[0]
		expectMsg := fmt.Sprintf("%s: %s", CACHE_FETCH_ERR, guid)
		if expectMsg != emsg {
			t.Errorf("cache fetch error expected '%s', got '%s'", expectMsg, emsg)
		}
	}
}

func runImportHelmFailureTests(t *testing.T, tests []*helmImportFailTest) {
	for _, td := range tests {
		req := makeImportHelmRequest(t, td.filename, td.appname, td.valuesname,
			false)
		eresult := doErrRequest(t, req)
		emsg := eresult.Data[0]
		if strings.Index(emsg, td.errmsg) == -1 {
			t.Error("missing expected error", emsg)
		}
	}
}

func TestImportHelm(t *testing.T) {
	successTests := []*helmImportTest{
		&helmImportTest{"sd", "m2hbundle-api-output.tgz", "sd_values.yaml"},
		&helmImportTest{"sd", "m2hbundle-api-output.tgz", "sd2_values.yaml"},
	}
	failTests := []*helmImportFailTest{
		&helmImportFailTest{"bad-values-file", "m2hbundle-api-output.tgz",
			"wrongvalues",
			"values file: 'wrongvalues' not found"},
		&helmImportFailTest{"bad-compression", "bday4.yml", "dummy",
			"can't gunzip decoded input"},
		&helmImportFailTest{"bad-tar-file", "bday4.gzip", "dummy",
			"error reading tar file"},
	}
	runImportHelmSuccessTests(t, successTests)
	runImportHelmFailureTests(t, failTests)
}
