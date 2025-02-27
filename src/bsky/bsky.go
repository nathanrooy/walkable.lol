package bsky

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"walkable/src/utils"
)

const PDS_URL string = "https://bsky.social"

type Records struct {
	Cursor  string   `json:"cursor"`
	Records []Record `json:"records"`
}

type Record struct {
	CID   string `json:"cid"`
	URI   string `json:"uri"`
	Value struct {
		CreatedAT string `json:"createdAt"`
	} `json:"value"`
}

type Session struct {
	DID       string `json:"did"`
	AccessJWT string `json:"accessJwt"`
}

type EmbedResponse struct {
	Blob Embed `json:"blob"`
}

type Embed struct {
	Type string `json:"$type"`
	Ref  struct {
		Link string `json:"$link"`
	} `json:"ref"`
	Mimetype string `json:"mimeType"`
	Size     int64  `json:"size"`
}

func _login() Session {
	var session Session
	postPayload := []byte(`{
		"identifier":"` + os.Getenv("BSKY_USER") + `",
		"password":"` + os.Getenv("BSKY_PSWD") + `"
	}`)

	req, err := http.NewRequest(
		"POST",
		PDS_URL+"/xrpc/com.atproto.server.createSession",
		bytes.NewBuffer(postPayload),
	)
	if err != nil {
		log.Println("> Error creating request:", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("> Error sending request:", err)
	}
	defer resp.Body.Close()

	log.Println("> BSKY login response code:", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("> Error reading response body:", err)
	}

	err = json.Unmarshal(body, &session)
	if err != nil {
		log.Println("> Error marshaling json response:", err)
	}

	return session
}

func _uploadImage(s Session, p utils.Post) Embed {
	var mimetype string = "image/png"

	req, err := http.NewRequest(
		"POST",
		PDS_URL+"/xrpc/com.atproto.repo.uploadBlob",
		&p.ImgBuf,
	)
	if err != nil {
		log.Println("> Error creating request:", err)
	}

	req.Header.Set("Content-Type", mimetype)
	req.Header.Set("Authorization", "Bearer "+s.AccessJWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("> Error sending request:", err)
	}
	defer resp.Body.Close()

	log.Println("> BSKY image upload response code:", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("> Error reading response body:", err)
	}

	var embedResp EmbedResponse
	err = json.Unmarshal(body, &embedResp)
	if err != nil {
		log.Println("> Error marshalling json response:", err)
	}

	return embedResp.Blob

}

func _post(s Session, p utils.Post) {

	embed := _uploadImage(s, p)
	dLen := len(p.Description)

	images := map[string]interface{}{
		"alt":   p.Description,
		"image": embed,
	}

	post := map[string]interface{}{
		"repo":       s.DID,
		"collection": "app.bsky.feed.post",
		"record": map[string]interface{}{
			"embed": map[string]interface{}{
				"$type":  "app.bsky.embed.images",
				"images": [1]map[string]interface{}{images},
			},
			"$type":     "app.bsky.feed.post",
			"text":      p.Description + "#suburbs #fail",
			"createdAt": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			"facets": []map[string]interface{}{
				{
					"index": map[string]int{
						"byteStart": dLen,
						"byteEnd":   dLen + 8,
					},
					"features": []map[string]string{
						{
							"$type": "app.bsky.richtext.facet#tag",
							"tag":   "suburbs",
						},
					},
				},
				{
					"index": map[string]int{
						"byteStart": dLen + 9,
						"byteEnd":   dLen + 14,
					},
					"features": []map[string]string{
						{
							"$type": "app.bsky.richtext.facet#tag",
							"tag":   "fail",
						},
					},
				},
			},
		},
	}

	postJSON, err := json.Marshal(post)
	if err != nil {
		log.Println("> Error marshalling JSON", err)
	}

	req, err := http.NewRequest(
		"POST",
		PDS_URL+"/xrpc/com.atproto.repo.createRecord",
		bytes.NewBuffer(postJSON),
	)
	if err != nil {
		log.Println("> Error creating request:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.AccessJWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("> Error sending request:", err)
	}
	defer resp.Body.Close()

	log.Println("> BSKY create post response code:", resp.StatusCode)

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("> Error reading response body:", err)
	}
}

func _deleteRecord(s Session, r Record) {

	log.Printf("> Purging: %v\n", r.URI)
	uri := strings.Split(r.URI, "/")
	rkey := uri[len(uri)-1]

	payload := map[string]string{
		"repo":       s.DID,
		"collection": "app.bsky.feed.post",
		"rkey":       rkey,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Println("> Error marshalling JSON", err)
	}

	req, err := http.NewRequest(
		"POST",
		PDS_URL+"/xrpc/com.atproto.repo.deleteRecord",
		bytes.NewBuffer(payloadJSON),
	)
	if err != nil {
		log.Println("> Error creating request:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.AccessJWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("> Error sending request:", err)
	}
	defer resp.Body.Close()

	log.Println("> BSKY delete post response code:", resp.StatusCode)

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("> Error reading response body:", err)
	}
}

func _purge(s Session) {

	u, err := url.Parse(PDS_URL + "/xrpc/com.atproto.repo.listRecords")
	if err != nil {
		log.Println("> Error parsing URL:", err)
	}

	q := u.Query()
	q.Set("repo", s.DID)
	q.Set("collection", "app.bsky.feed.post")
	q.Set("limit", "25")
	q.Set("reverse", "true") // only get the 25 oldest posts
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Println("> Error creating request:", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("> Error sending request:", err)
	}
	defer resp.Body.Close()

	log.Println("> BSKY listRecords response code:", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("> Error reading response body:", err)
	}

	var records Records
	err = json.Unmarshal(body, &records)
	if err != nil {
		log.Println("> Error marshalling json response:", err)
	}

	for i := 0; i < len(records.Records); i++ {
		t_post, _ := time.Parse(time.RFC3339, records.Records[i].Value.CreatedAT)
		t_delta := time.Now().UTC().Sub(t_post).Seconds()
		if t_delta >= 90*24*3600 {
			_deleteRecord(s, records.Records[i])
		}
	}
}

func Run(p utils.Post) {

	s := _login()
	_post(s, p)
	_purge(s)

}
