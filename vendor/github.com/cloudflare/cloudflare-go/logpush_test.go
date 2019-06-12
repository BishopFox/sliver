package cloudflare

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"

)

const (
	jobID                       = 1
	serverLogpushJobDescription = `{
    "id": %d,
    "enabled": false,
	"name": "example.com",
    "logpull_options": "fields=RayID,ClientIP,EdgeStartTimestamp&timestamps=rfc3339",
	"destination_conf": "s3://mybucket/logs?region=us-west-2",
	"last_complete": "%[2]s",
	"last_error": "%[2]s",
	"error_message": "test"
  }
`
	serverLogpushGetOwnershipChallengeDescription = `{
    "filename": "logs/challenge-filename.txt",
	"valid": true,
	"message": ""
  }
`
)

var (
	testLogpushTimestamp     = time.Now().UTC()
	expectedLogpushJobStruct = LogpushJob{
		ID:              jobID,
		Enabled:         false,
		Name:            "example.com",
		LogpullOptions:  "fields=RayID,ClientIP,EdgeStartTimestamp&timestamps=rfc3339",
		DestinationConf: "s3://mybucket/logs?region=us-west-2",
		LastComplete:    &testLogpushTimestamp,
		LastError:       &testLogpushTimestamp,
		ErrorMessage:    "test",
	}
	expectedLogpushGetOwnershipChallengeStruct = LogpushGetOwnershipChallenge{
		Filename: "logs/challenge-filename.txt",
		Valid:    true,
		Message:  "",
	}
	expectedUpdatedLogpushJobStruct = LogpushJob{
		ID: jobID, 
		Enabled: true, 
		Name: "updated.com", 
		LogpullOptions: "fields=RayID,ClientIP,EdgeStartTimestamp", 
		DestinationConf: "gs://mybucket/logs", 
		LastComplete: &testLogpushTimestamp, 
		LastError: &testLogpushTimestamp, 
		ErrorMessage: "test",
	}
)

func TestLogpushJobs(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": [
			%s
		  ],
		  "success": true,
		  "errors": null,
		  "messages": null,
		  "result_info": {
			"page": 1,
			"per_page": 25,
			"count": 1,
			"total_count": 1
		  }
		}
		`, fmt.Sprintf(serverLogpushJobDescription, jobID, testLogpushTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/jobs", handler)
	want := []LogpushJob{expectedLogpushJobStruct}

	actual, err := client.LogpushJobs(testZoneID)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestGetLogpushJob(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, fmt.Sprintf(serverLogpushJobDescription, jobID, testLogpushTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/jobs/"+strconv.Itoa(jobID), handler)
	want := expectedLogpushJobStruct

	actual, err := client.LogpushJob(testZoneID, jobID)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreateLogpushJob(t *testing.T) {
	setup()
	defer teardown()
	newJob := LogpushJob{
		Enabled:         false,
		Name:            "example.com",
		LogpullOptions:  "fields=RayID,ClientIP,EdgeStartTimestamp&timestamps=rfc3339",
		DestinationConf: "s3://mybucket/logs?region=us-west-2",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, fmt.Sprintf(serverLogpushJobDescription, jobID, testLogpushTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/jobs", handler)
	want := &expectedLogpushJobStruct

	actual, err := client.CreateLogpushJob(testZoneID, newJob)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateLogpushJob(t *testing.T) {
	setup()
	defer teardown()
	updatedJob := LogpushJob{
		Enabled: true, 
		Name: "updated.com", 
		LogpullOptions: "fields=RayID,ClientIP,EdgeStartTimestamp", 
		DestinationConf: "gs://mybucket/logs",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, fmt.Sprintf(serverLogpushJobDescription, jobID, testLogpushTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/jobs/"+strconv.Itoa(jobID), handler)

	err := client.UpdateLogpushJob(testZoneID, jobID, updatedJob)
	assert.NoError(t, err)
}

func TestDeleteLogpushJob(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "DELETE", "Expected method 'DELETE', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
		  "result": null,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/jobs/"+strconv.Itoa(jobID), handler)

	err := client.DeleteLogpushJob(testZoneID, jobID)
	assert.NoError(t, err)
}

func TestGetLogpushOwnershipChallenge(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, serverLogpushGetOwnershipChallengeDescription)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/logpush/ownership", handler)

	want := &expectedLogpushGetOwnershipChallengeStruct

	actual, err := client.GetLogpushOwnershipChallenge(testZoneID, "destination_conf")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestValidateLogpushOwnershipChallenge(t *testing.T) {
	testCases := map[string]struct {
		isValid bool
	}{
		"ownership is valid": {
			isValid: true,
		},
		"ownership is not valid": {
			isValid: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			setup()
			defer teardown()

			handler := func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
				w.Header().Set("content-type", "application/json")
				fmt.Fprintf(w, `{
				  "result": {
					"valid": %v
				  },
				  "success": true,
				  "errors": null,
				  "messages": null
				}
				`, tc.isValid)
			}

			mux.HandleFunc("/zones/"+testZoneID+"/logpush/ownership/validate", handler)

			actual, err := client.ValidateLogpushOwnershipChallenge(testZoneID, "destination_conf", "ownership_challenge")
			if assert.NoError(t, err) {
				assert.Equal(t, tc.isValid, actual)
			}
		})
	}
}

func TestCheckLogpushDestiantionExists(t *testing.T) {
	testCases := map[string]struct {
		exists bool
	}{
		"destination exists": {
			exists: true,
		},
		"destination does not exists": {
			exists: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			setup()
			defer teardown()

			handler := func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
				w.Header().Set("content-type", "application/json")
				fmt.Fprintf(w, `{
				  "result": {
					"exists": %v
				  },
				  "success": true,
				  "errors": null,
				  "messages": null
				}
				`, tc.exists)
			}

			mux.HandleFunc("/zones/"+testZoneID+"/logpush/validate/destination/exists", handler)

			actual, err := client.CheckLogpushDestinationExists(testZoneID, "destination_conf")
			if assert.NoError(t, err) {
				assert.Equal(t, tc.exists, actual)
			}
		})
	}
}
