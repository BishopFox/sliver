package viber

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// PostData to viber API
func (v *Viber) PostData(url string, i interface{}) ([]byte, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	Log.Println("Post data:", string(b))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	req.Header.Add("X-Viber-Auth-Token", v.AppKey)
	req.Close = true
	
	if v.client == nil {
		v.client = &http.Client{}
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
