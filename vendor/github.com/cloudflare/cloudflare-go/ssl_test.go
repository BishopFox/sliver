package cloudflare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateSSL(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method, "Expected method 'POST', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"certificate":"-----BEGIN CERTIFICATE----- MIIDtTCCAp2gAwIBAgIJAM15n7fdxhRtMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV BAYTAlVTMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX aWRnaXRzIFB0eSBMdGQwHhcNMTQwMzExMTkyMTU5WhcNMTQwNDEwMTkyMTU5WjBF MQswCQYDVQQGEwJVUzETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50 ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB CgKCAQEAvq3sKsHpeduJHimOK+fvQdKsI8z8A05MZyyLp2/R/GE8FjNv+hkVY1WQ LIyTNNQH7CJecE1nbTfo8Y56S7x/rhxC6/DJ8MIulapFPnorq46KU6yRxiM0MQ3N nTJHlHA2ozZta6YBBfVfhHWl1F0IfNbXCLKvGwWWMbCx43OfW6KTkbRnE6gFWKuO fSO5h2u5TaWVuSIzBvYs7Vza6m+gtYAvKAJV2nSZ+eSEFPDo29corOy8+huEOUL8 5FAw4BFPsr1TlrlGPFitduQUHGrSL7skk1ESGza0to3bOtrodKei2s9bk5MXm7lZ qI+WZJX4Zu9+mzZhc9pCVi8r/qlXuQIDAQABo4GnMIGkMB0GA1UdDgQWBBRvavf+ sWM4IwKiH9X9w1vl6nUVRDB1BgNVHSMEbjBsgBRvavf+sWM4IwKiH9X9w1vl6nUV RKFJpEcwRTELMAkGA1UEBhMCVVMxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNV BAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAM15n7fdxhRtMAwGA1UdEwQF MAMBAf8wDQYJKoZIhvcNAQEFBQADggEBABY2ZzBaW0dMsAAT7tPJzrVWVzQx6KU4 UEBLudIlWPlkAwTnINCWR/8eNjCCmGA4heUdHmazdpPa8RzwOmc0NT1NQqzSyktt vTqb4iHD7+8f9MqJ9/FssCfTtqr/Qst/hGH4Wmdf1EJ/6FqYAAb5iRlPgshFZxU8 uXtA8hWn6fK6eISD9HBdcAFToUvKNZ1BIDPvh9f95Ine8ar6yGd56TUNrHR8eHBs ESxz5ddVR/oWRysNJ+aGAyYqHS8S/ttmC7r4XCAHqXptkHPCGRqkAhsterYhd4I8 /cBzejUobNCjjHFbtkAL/SjxZOLW+pNkZwfeYdM8iPkD54Uua1v2tdw= -----END CERTIFICATE-----","private_key":"-----BEGIN RSA PRIVATE KEY-----MIIEowIBAAKCAQEAl 1cSc0vfcJLI4ZdWjiZZqy86Eof4czCwilyjXdvHqbdgDjz9H6K/0FX78EzVdfyExESptPCDl5YYjvcZyAWlgNfYEpFpGeoh/pTFW3hlyKImh4EgBXbDrR251J Ew2Nf56X3duibI6X20gKZA6cvdmWeKh MOOXuh1bSPU3dkb4YOF/fng5iGrx0q3txdMQXTPMZ1uXHFcBH7idgViYesXUBhdll3GP1N Y8laq0yrqh 8HMsZK m27MebqonbNmjOqE218lVEvjCdRO6xvNXrO6vNJBoGn2eGwZ8BVd0mTA3Tj43/2cmxQFY9FLq56cCXqYI1fbRRib ZLrjSNkwIDAQABAoIBABfAjjsjjxc0NxcYvKOMUb9Rpj8Sx6U/o/tDC5u XmsGX37aaJmC5yw9BQiAxgvXtQryEl5uoNoqOdsxzKV6yM0vPcwKEJVBd4G6yx6AjVJZnc2qf72erR7BbA2CQh scMDRBKE041HhgTBRNP6roim0SOgYP5JZIrGAQXNIkyE0fZc5gZNUt388ne/mjWM6Xi08BDGurLC68nsdt7Nd UYqeBVxo2EqChp5vKYZYEcG8h9XBj4u4NIwg1Mty2JqX30uBjoHvF5w/pMs8lG uvj6JR9I 19wtCuccbAJl 4cUq03UQoIDmwejea oC8A8WJr3vVpODDWrvAsjllGPBECgYEAyQRa6edYO6bsSvgbM13qXW9OQTn9YmgzfN24Ux1D66TQU6sBSLdfSHshDhTCi Ax 698aJNRWujAakA2DDgspSx98aRnHbF zvY7i7iWGesN6uN0zL 6/MK5uWoieGZRjgk230fLk00l4/FK1mJIp0apr0Lis9xmDjP5AaUPTUUCgYEAwXuhTHZWPT6v8YwOksjbuK UDkIIvyMux53kb73vrkgMboS4DB1zMLNyG 9EghS414CFROUwGl4ZUKboH1Jo5G34y8VgDuHjirTqL2H6 zNpML iMrWCXjpFKkxwPbeQnEAZ 5Rud4d PTyXAt71blZHE9tZ4KHy8cU1iKc9APcCgYAIqKZd4vg7AZK2G//X85iv06aUSrIudfyZyVcyRVVyphPPNtOEVVnGXn9rAtvqeIrOo52BR68 cj4vlXp hkDuEH QVBuY/NdQhOzFtPrKPQTJdGjIlQ2x65Vidj7r3sRukNkLPyV2v D885zcpTkp83JFuWTYiIrg275DIuAI3QKBgAglM0IrzS g3vlVQxvM1ussgRgkkYeybHq82 wUW 3DXLqeXb0s1DedplUkuoabZriz0Wh4GZFSmtA5ZpZC uV697lkYsndmp2xRhaekllW7bu pY5q88URwO2p8CO5AZ6CWFWuBwSDML5VOapGRqDRgwaD oGpb7fb7IgHOls7AoGBAJnL6Q8t35uYJ8J8hY7wso88IE04z6VaT8WganxcndesWER9eFQDHDDy//ZYeyt6M41uIY CL Vkm9Kwl/bHLJKdnOE1a9NdE6mtfah0Bk2u/YOuzyu5mmcgZiX X/OZuEbGmmbZOR1FCuIyrNYfwYohhcZP7/r0Ia/1GpkHc3Bi-----END RSA PRIVATE KEY-----","bundle_method":"ubiquitous"}`, string(b))
		}

		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
          "success": true,
          "errors": [],
          "messages": [],
          "result": {
            "id": "7e7b8deba8538af625850b7b2530034c",
            "hosts": [
              "example.com"
            ],
            "issuer": "GlobalSign",
            "signature": "SHA256WithRSA",
            "status": "active",
            "bundle_method": "ubiquitous",
            "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
            "uploaded_on": "2014-01-01T05:20:00Z",
            "modified_on": "2014-01-01T05:20:00Z",
            "expires_on": "2016-01-01T05:20:00Z",
            "priority": 1
          }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates", handler)

	hosts := make([]string, 1, 4)
	hosts[0] = "example.com"
	uploadedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2016-01-01T05:20:00Z")
	want := ZoneCustomSSL{
		ID:           "7e7b8deba8538af625850b7b2530034c",
		Hosts:        hosts,
		Issuer:       "GlobalSign",
		Signature:    "SHA256WithRSA",
		Status:       "active",
		BundleMethod: "ubiquitous",
		ZoneID:       "023e105f4ecef8ad9ca31a8372d0c353",
		UploadedOn:   uploadedOn,
		ModifiedOn:   modifiedOn,
		ExpiresOn:    expiresOn,
		Priority:     1,
	}

	actual, err := client.CreateSSL("023e105f4ecef8ad9ca31a8372d0c353", ZoneCustomSSLOptions{
		Certificate:  "-----BEGIN CERTIFICATE----- MIIDtTCCAp2gAwIBAgIJAM15n7fdxhRtMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV BAYTAlVTMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX aWRnaXRzIFB0eSBMdGQwHhcNMTQwMzExMTkyMTU5WhcNMTQwNDEwMTkyMTU5WjBF MQswCQYDVQQGEwJVUzETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50 ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB CgKCAQEAvq3sKsHpeduJHimOK+fvQdKsI8z8A05MZyyLp2/R/GE8FjNv+hkVY1WQ LIyTNNQH7CJecE1nbTfo8Y56S7x/rhxC6/DJ8MIulapFPnorq46KU6yRxiM0MQ3N nTJHlHA2ozZta6YBBfVfhHWl1F0IfNbXCLKvGwWWMbCx43OfW6KTkbRnE6gFWKuO fSO5h2u5TaWVuSIzBvYs7Vza6m+gtYAvKAJV2nSZ+eSEFPDo29corOy8+huEOUL8 5FAw4BFPsr1TlrlGPFitduQUHGrSL7skk1ESGza0to3bOtrodKei2s9bk5MXm7lZ qI+WZJX4Zu9+mzZhc9pCVi8r/qlXuQIDAQABo4GnMIGkMB0GA1UdDgQWBBRvavf+ sWM4IwKiH9X9w1vl6nUVRDB1BgNVHSMEbjBsgBRvavf+sWM4IwKiH9X9w1vl6nUV RKFJpEcwRTELMAkGA1UEBhMCVVMxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNV BAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAM15n7fdxhRtMAwGA1UdEwQF MAMBAf8wDQYJKoZIhvcNAQEFBQADggEBABY2ZzBaW0dMsAAT7tPJzrVWVzQx6KU4 UEBLudIlWPlkAwTnINCWR/8eNjCCmGA4heUdHmazdpPa8RzwOmc0NT1NQqzSyktt vTqb4iHD7+8f9MqJ9/FssCfTtqr/Qst/hGH4Wmdf1EJ/6FqYAAb5iRlPgshFZxU8 uXtA8hWn6fK6eISD9HBdcAFToUvKNZ1BIDPvh9f95Ine8ar6yGd56TUNrHR8eHBs ESxz5ddVR/oWRysNJ+aGAyYqHS8S/ttmC7r4XCAHqXptkHPCGRqkAhsterYhd4I8 /cBzejUobNCjjHFbtkAL/SjxZOLW+pNkZwfeYdM8iPkD54Uua1v2tdw= -----END CERTIFICATE-----",
		PrivateKey:   "-----BEGIN RSA PRIVATE KEY-----MIIEowIBAAKCAQEAl 1cSc0vfcJLI4ZdWjiZZqy86Eof4czCwilyjXdvHqbdgDjz9H6K/0FX78EzVdfyExESptPCDl5YYjvcZyAWlgNfYEpFpGeoh/pTFW3hlyKImh4EgBXbDrR251J Ew2Nf56X3duibI6X20gKZA6cvdmWeKh MOOXuh1bSPU3dkb4YOF/fng5iGrx0q3txdMQXTPMZ1uXHFcBH7idgViYesXUBhdll3GP1N Y8laq0yrqh 8HMsZK m27MebqonbNmjOqE218lVEvjCdRO6xvNXrO6vNJBoGn2eGwZ8BVd0mTA3Tj43/2cmxQFY9FLq56cCXqYI1fbRRib ZLrjSNkwIDAQABAoIBABfAjjsjjxc0NxcYvKOMUb9Rpj8Sx6U/o/tDC5u XmsGX37aaJmC5yw9BQiAxgvXtQryEl5uoNoqOdsxzKV6yM0vPcwKEJVBd4G6yx6AjVJZnc2qf72erR7BbA2CQh scMDRBKE041HhgTBRNP6roim0SOgYP5JZIrGAQXNIkyE0fZc5gZNUt388ne/mjWM6Xi08BDGurLC68nsdt7Nd UYqeBVxo2EqChp5vKYZYEcG8h9XBj4u4NIwg1Mty2JqX30uBjoHvF5w/pMs8lG uvj6JR9I 19wtCuccbAJl 4cUq03UQoIDmwejea oC8A8WJr3vVpODDWrvAsjllGPBECgYEAyQRa6edYO6bsSvgbM13qXW9OQTn9YmgzfN24Ux1D66TQU6sBSLdfSHshDhTCi Ax 698aJNRWujAakA2DDgspSx98aRnHbF zvY7i7iWGesN6uN0zL 6/MK5uWoieGZRjgk230fLk00l4/FK1mJIp0apr0Lis9xmDjP5AaUPTUUCgYEAwXuhTHZWPT6v8YwOksjbuK UDkIIvyMux53kb73vrkgMboS4DB1zMLNyG 9EghS414CFROUwGl4ZUKboH1Jo5G34y8VgDuHjirTqL2H6 zNpML iMrWCXjpFKkxwPbeQnEAZ 5Rud4d PTyXAt71blZHE9tZ4KHy8cU1iKc9APcCgYAIqKZd4vg7AZK2G//X85iv06aUSrIudfyZyVcyRVVyphPPNtOEVVnGXn9rAtvqeIrOo52BR68 cj4vlXp hkDuEH QVBuY/NdQhOzFtPrKPQTJdGjIlQ2x65Vidj7r3sRukNkLPyV2v D885zcpTkp83JFuWTYiIrg275DIuAI3QKBgAglM0IrzS g3vlVQxvM1ussgRgkkYeybHq82 wUW 3DXLqeXb0s1DedplUkuoabZriz0Wh4GZFSmtA5ZpZC uV697lkYsndmp2xRhaekllW7bu pY5q88URwO2p8CO5AZ6CWFWuBwSDML5VOapGRqDRgwaD oGpb7fb7IgHOls7AoGBAJnL6Q8t35uYJ8J8hY7wso88IE04z6VaT8WganxcndesWER9eFQDHDDy//ZYeyt6M41uIY CL Vkm9Kwl/bHLJKdnOE1a9NdE6mtfah0Bk2u/YOuzyu5mmcgZiX X/OZuEbGmmbZOR1FCuIyrNYfwYohhcZP7/r0Ia/1GpkHc3Bi-----END RSA PRIVATE KEY-----",
		BundleMethod: "ubiquitous",
	})
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.CreateSSL("bar", ZoneCustomSSLOptions{})
	assert.Error(t, err)
}

func TestListSSL(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
          "success": true,
          "errors": [],
          "messages": [],
          "result": [
            {
              "id": "7e7b8deba8538af625850b7b2530034c",
              "hosts": [
                "example.com"
              ],
              "issuer": "GlobalSign",
              "signature": "SHA256WithRSA",
              "status": "active",
              "bundle_method": "ubiquitous",
              "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
              "uploaded_on": "2014-01-01T05:20:00Z",
              "modified_on": "2014-01-01T05:20:00Z",
              "expires_on": "2016-01-01T05:20:00Z",
              "priority": 1
            }
          ],
          "result_info": {
            "page": 1,
            "per_page": 20,
            "count": 1,
            "total_count": 2000
          }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates", handler)

	hosts := make([]string, 1, 4)
	hosts[0] = "example.com"
	uploadedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2016-01-01T05:20:00Z")

	want := make([]ZoneCustomSSL, 1, 4)
	want[0] = ZoneCustomSSL{
		ID:           "7e7b8deba8538af625850b7b2530034c",
		Hosts:        hosts,
		Issuer:       "GlobalSign",
		Signature:    "SHA256WithRSA",
		Status:       "active",
		BundleMethod: "ubiquitous",
		ZoneID:       "023e105f4ecef8ad9ca31a8372d0c353",
		UploadedOn:   uploadedOn,
		ModifiedOn:   modifiedOn,
		ExpiresOn:    expiresOn,
		Priority:     1,
	}

	actual, err := client.ListSSL("023e105f4ecef8ad9ca31a8372d0c353")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.ListSSL("bar")
	assert.Error(t, err)
}

func TestSSLDetails(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
          "success": true,
          "errors": [],
          "messages": [],
          "result": {
            "id": "7e7b8deba8538af625850b7b2530034c",
            "hosts": [
              "example.com"
            ],
            "issuer": "GlobalSign",
            "signature": "SHA256WithRSA",
            "status": "active",
            "bundle_method": "ubiquitous",
            "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
            "uploaded_on": "2014-01-01T05:20:00Z",
            "modified_on": "2014-01-01T05:20:00Z",
            "expires_on": "2016-01-01T05:20:00Z",
            "priority": 1
          }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates/7e7b8deba8538af625850b7b2530034c", handler)

	hosts := make([]string, 1, 4)
	hosts[0] = "example.com"
	uploadedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2016-01-01T05:20:00Z")
	want := ZoneCustomSSL{
		ID:           "7e7b8deba8538af625850b7b2530034c",
		Hosts:        hosts,
		Issuer:       "GlobalSign",
		Signature:    "SHA256WithRSA",
		Status:       "active",
		BundleMethod: "ubiquitous",
		ZoneID:       "023e105f4ecef8ad9ca31a8372d0c353",
		UploadedOn:   uploadedOn,
		ModifiedOn:   modifiedOn,
		ExpiresOn:    expiresOn,
		Priority:     1,
	}

	actual, err := client.SSLDetails("023e105f4ecef8ad9ca31a8372d0c353", "7e7b8deba8538af625850b7b2530034c")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.SSLDetails("023e105f4ecef8ad9ca31a8372d0c353", "bar")
	assert.Error(t, err)
}

func TestUpdateSSL(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method, "Expected method 'PATCH', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"certificate":"-----BEGIN CERTIFICATE----- MIIDtTCCAp2gAwIBAgIJAM15n7fdxhRtMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV BAYTAlVTMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX aWRnaXRzIFB0eSBMdGQwHhcNMTQwMzExMTkyMTU5WhcNMTQwNDEwMTkyMTU5WjBF MQswCQYDVQQGEwJVUzETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50 ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB CgKCAQEAvq3sKsHpeduJHimOK+fvQdKsI8z8A05MZyyLp2/R/GE8FjNv+hkVY1WQ LIyTNNQH7CJecE1nbTfo8Y56S7x/rhxC6/DJ8MIulapFPnorq46KU6yRxiM0MQ3N nTJHlHA2ozZta6YBBfVfhHWl1F0IfNbXCLKvGwWWMbCx43OfW6KTkbRnE6gFWKuO fSO5h2u5TaWVuSIzBvYs7Vza6m+gtYAvKAJV2nSZ+eSEFPDo29corOy8+huEOUL8 5FAw4BFPsr1TlrlGPFitduQUHGrSL7skk1ESGza0to3bOtrodKei2s9bk5MXm7lZ qI+WZJX4Zu9+mzZhc9pCVi8r/qlXuQIDAQABo4GnMIGkMB0GA1UdDgQWBBRvavf+ sWM4IwKiH9X9w1vl6nUVRDB1BgNVHSMEbjBsgBRvavf+sWM4IwKiH9X9w1vl6nUV RKFJpEcwRTELMAkGA1UEBhMCVVMxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNV BAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAM15n7fdxhRtMAwGA1UdEwQF MAMBAf8wDQYJKoZIhvcNAQEFBQADggEBABY2ZzBaW0dMsAAT7tPJzrVWVzQx6KU4 UEBLudIlWPlkAwTnINCWR/8eNjCCmGA4heUdHmazdpPa8RzwOmc0NT1NQqzSyktt vTqb4iHD7+8f9MqJ9/FssCfTtqr/Qst/hGH4Wmdf1EJ/6FqYAAb5iRlPgshFZxU8 uXtA8hWn6fK6eISD9HBdcAFToUvKNZ1BIDPvh9f95Ine8ar6yGd56TUNrHR8eHBs ESxz5ddVR/oWRysNJ+aGAyYqHS8S/ttmC7r4XCAHqXptkHPCGRqkAhsterYhd4I8 /cBzejUobNCjjHFbtkAL/SjxZOLW+pNkZwfeYdM8iPkD54Uua1v2tdw= -----END CERTIFICATE-----","private_key":"-----BEGIN RSA PRIVATE KEY-----MIIEowIBAAKCAQEAl 1cSc0vfcJLI4ZdWjiZZqy86Eof4czCwilyjXdvHqbdgDjz9H6K/0FX78EzVdfyExESptPCDl5YYjvcZyAWlgNfYEpFpGeoh/pTFW3hlyKImh4EgBXbDrR251J Ew2Nf56X3duibI6X20gKZA6cvdmWeKh MOOXuh1bSPU3dkb4YOF/fng5iGrx0q3txdMQXTPMZ1uXHFcBH7idgViYesXUBhdll3GP1N Y8laq0yrqh 8HMsZK m27MebqonbNmjOqE218lVEvjCdRO6xvNXrO6vNJBoGn2eGwZ8BVd0mTA3Tj43/2cmxQFY9FLq56cCXqYI1fbRRib ZLrjSNkwIDAQABAoIBABfAjjsjjxc0NxcYvKOMUb9Rpj8Sx6U/o/tDC5u XmsGX37aaJmC5yw9BQiAxgvXtQryEl5uoNoqOdsxzKV6yM0vPcwKEJVBd4G6yx6AjVJZnc2qf72erR7BbA2CQh scMDRBKE041HhgTBRNP6roim0SOgYP5JZIrGAQXNIkyE0fZc5gZNUt388ne/mjWM6Xi08BDGurLC68nsdt7Nd UYqeBVxo2EqChp5vKYZYEcG8h9XBj4u4NIwg1Mty2JqX30uBjoHvF5w/pMs8lG uvj6JR9I 19wtCuccbAJl 4cUq03UQoIDmwejea oC8A8WJr3vVpODDWrvAsjllGPBECgYEAyQRa6edYO6bsSvgbM13qXW9OQTn9YmgzfN24Ux1D66TQU6sBSLdfSHshDhTCi Ax 698aJNRWujAakA2DDgspSx98aRnHbF zvY7i7iWGesN6uN0zL 6/MK5uWoieGZRjgk230fLk00l4/FK1mJIp0apr0Lis9xmDjP5AaUPTUUCgYEAwXuhTHZWPT6v8YwOksjbuK UDkIIvyMux53kb73vrkgMboS4DB1zMLNyG 9EghS414CFROUwGl4ZUKboH1Jo5G34y8VgDuHjirTqL2H6 zNpML iMrWCXjpFKkxwPbeQnEAZ 5Rud4d PTyXAt71blZHE9tZ4KHy8cU1iKc9APcCgYAIqKZd4vg7AZK2G//X85iv06aUSrIudfyZyVcyRVVyphPPNtOEVVnGXn9rAtvqeIrOo52BR68 cj4vlXp hkDuEH QVBuY/NdQhOzFtPrKPQTJdGjIlQ2x65Vidj7r3sRukNkLPyV2v D885zcpTkp83JFuWTYiIrg275DIuAI3QKBgAglM0IrzS g3vlVQxvM1ussgRgkkYeybHq82 wUW 3DXLqeXb0s1DedplUkuoabZriz0Wh4GZFSmtA5ZpZC uV697lkYsndmp2xRhaekllW7bu pY5q88URwO2p8CO5AZ6CWFWuBwSDML5VOapGRqDRgwaD oGpb7fb7IgHOls7AoGBAJnL6Q8t35uYJ8J8hY7wso88IE04z6VaT8WganxcndesWER9eFQDHDDy//ZYeyt6M41uIY CL Vkm9Kwl/bHLJKdnOE1a9NdE6mtfah0Bk2u/YOuzyu5mmcgZiX X/OZuEbGmmbZOR1FCuIyrNYfwYohhcZP7/r0Ia/1GpkHc3Bi-----END RSA PRIVATE KEY-----","bundle_method":"ubiquitous"}`, string(b))
		}

		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
          "success": true,
          "errors": [],
          "messages": [],
          "result": {
            "id": "7e7b8deba8538af625850b7b2530034c",
            "hosts": [
              "example.com"
            ],
            "issuer": "GlobalSign",
            "signature": "SHA256WithRSA",
            "status": "active",
            "bundle_method": "ubiquitous",
            "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
            "uploaded_on": "2014-01-01T05:20:00Z",
            "modified_on": "2014-01-01T05:20:00Z",
            "expires_on": "2016-01-01T05:20:00Z",
            "priority": 1
          }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates/7e7b8deba8538af625850b7b2530034c", handler)

	hosts := make([]string, 1, 4)
	hosts[0] = "example.com"
	uploadedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2016-01-01T05:20:00Z")
	want := ZoneCustomSSL{
		ID:           "7e7b8deba8538af625850b7b2530034c",
		Hosts:        hosts,
		Issuer:       "GlobalSign",
		Signature:    "SHA256WithRSA",
		Status:       "active",
		BundleMethod: "ubiquitous",
		ZoneID:       "023e105f4ecef8ad9ca31a8372d0c353",
		UploadedOn:   uploadedOn,
		ModifiedOn:   modifiedOn,
		ExpiresOn:    expiresOn,
		Priority:     1,
	}

	actual, err := client.UpdateSSL("023e105f4ecef8ad9ca31a8372d0c353", "7e7b8deba8538af625850b7b2530034c", ZoneCustomSSLOptions{
		Certificate:  "-----BEGIN CERTIFICATE----- MIIDtTCCAp2gAwIBAgIJAM15n7fdxhRtMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV BAYTAlVTMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX aWRnaXRzIFB0eSBMdGQwHhcNMTQwMzExMTkyMTU5WhcNMTQwNDEwMTkyMTU5WjBF MQswCQYDVQQGEwJVUzETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50 ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB CgKCAQEAvq3sKsHpeduJHimOK+fvQdKsI8z8A05MZyyLp2/R/GE8FjNv+hkVY1WQ LIyTNNQH7CJecE1nbTfo8Y56S7x/rhxC6/DJ8MIulapFPnorq46KU6yRxiM0MQ3N nTJHlHA2ozZta6YBBfVfhHWl1F0IfNbXCLKvGwWWMbCx43OfW6KTkbRnE6gFWKuO fSO5h2u5TaWVuSIzBvYs7Vza6m+gtYAvKAJV2nSZ+eSEFPDo29corOy8+huEOUL8 5FAw4BFPsr1TlrlGPFitduQUHGrSL7skk1ESGza0to3bOtrodKei2s9bk5MXm7lZ qI+WZJX4Zu9+mzZhc9pCVi8r/qlXuQIDAQABo4GnMIGkMB0GA1UdDgQWBBRvavf+ sWM4IwKiH9X9w1vl6nUVRDB1BgNVHSMEbjBsgBRvavf+sWM4IwKiH9X9w1vl6nUV RKFJpEcwRTELMAkGA1UEBhMCVVMxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNV BAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAM15n7fdxhRtMAwGA1UdEwQF MAMBAf8wDQYJKoZIhvcNAQEFBQADggEBABY2ZzBaW0dMsAAT7tPJzrVWVzQx6KU4 UEBLudIlWPlkAwTnINCWR/8eNjCCmGA4heUdHmazdpPa8RzwOmc0NT1NQqzSyktt vTqb4iHD7+8f9MqJ9/FssCfTtqr/Qst/hGH4Wmdf1EJ/6FqYAAb5iRlPgshFZxU8 uXtA8hWn6fK6eISD9HBdcAFToUvKNZ1BIDPvh9f95Ine8ar6yGd56TUNrHR8eHBs ESxz5ddVR/oWRysNJ+aGAyYqHS8S/ttmC7r4XCAHqXptkHPCGRqkAhsterYhd4I8 /cBzejUobNCjjHFbtkAL/SjxZOLW+pNkZwfeYdM8iPkD54Uua1v2tdw= -----END CERTIFICATE-----",
		PrivateKey:   "-----BEGIN RSA PRIVATE KEY-----MIIEowIBAAKCAQEAl 1cSc0vfcJLI4ZdWjiZZqy86Eof4czCwilyjXdvHqbdgDjz9H6K/0FX78EzVdfyExESptPCDl5YYjvcZyAWlgNfYEpFpGeoh/pTFW3hlyKImh4EgBXbDrR251J Ew2Nf56X3duibI6X20gKZA6cvdmWeKh MOOXuh1bSPU3dkb4YOF/fng5iGrx0q3txdMQXTPMZ1uXHFcBH7idgViYesXUBhdll3GP1N Y8laq0yrqh 8HMsZK m27MebqonbNmjOqE218lVEvjCdRO6xvNXrO6vNJBoGn2eGwZ8BVd0mTA3Tj43/2cmxQFY9FLq56cCXqYI1fbRRib ZLrjSNkwIDAQABAoIBABfAjjsjjxc0NxcYvKOMUb9Rpj8Sx6U/o/tDC5u XmsGX37aaJmC5yw9BQiAxgvXtQryEl5uoNoqOdsxzKV6yM0vPcwKEJVBd4G6yx6AjVJZnc2qf72erR7BbA2CQh scMDRBKE041HhgTBRNP6roim0SOgYP5JZIrGAQXNIkyE0fZc5gZNUt388ne/mjWM6Xi08BDGurLC68nsdt7Nd UYqeBVxo2EqChp5vKYZYEcG8h9XBj4u4NIwg1Mty2JqX30uBjoHvF5w/pMs8lG uvj6JR9I 19wtCuccbAJl 4cUq03UQoIDmwejea oC8A8WJr3vVpODDWrvAsjllGPBECgYEAyQRa6edYO6bsSvgbM13qXW9OQTn9YmgzfN24Ux1D66TQU6sBSLdfSHshDhTCi Ax 698aJNRWujAakA2DDgspSx98aRnHbF zvY7i7iWGesN6uN0zL 6/MK5uWoieGZRjgk230fLk00l4/FK1mJIp0apr0Lis9xmDjP5AaUPTUUCgYEAwXuhTHZWPT6v8YwOksjbuK UDkIIvyMux53kb73vrkgMboS4DB1zMLNyG 9EghS414CFROUwGl4ZUKboH1Jo5G34y8VgDuHjirTqL2H6 zNpML iMrWCXjpFKkxwPbeQnEAZ 5Rud4d PTyXAt71blZHE9tZ4KHy8cU1iKc9APcCgYAIqKZd4vg7AZK2G//X85iv06aUSrIudfyZyVcyRVVyphPPNtOEVVnGXn9rAtvqeIrOo52BR68 cj4vlXp hkDuEH QVBuY/NdQhOzFtPrKPQTJdGjIlQ2x65Vidj7r3sRukNkLPyV2v D885zcpTkp83JFuWTYiIrg275DIuAI3QKBgAglM0IrzS g3vlVQxvM1ussgRgkkYeybHq82 wUW 3DXLqeXb0s1DedplUkuoabZriz0Wh4GZFSmtA5ZpZC uV697lkYsndmp2xRhaekllW7bu pY5q88URwO2p8CO5AZ6CWFWuBwSDML5VOapGRqDRgwaD oGpb7fb7IgHOls7AoGBAJnL6Q8t35uYJ8J8hY7wso88IE04z6VaT8WganxcndesWER9eFQDHDDy//ZYeyt6M41uIY CL Vkm9Kwl/bHLJKdnOE1a9NdE6mtfah0Bk2u/YOuzyu5mmcgZiX X/OZuEbGmmbZOR1FCuIyrNYfwYohhcZP7/r0Ia/1GpkHc3Bi-----END RSA PRIVATE KEY-----",
		BundleMethod: "ubiquitous",
	})
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.UpdateSSL("023e105f4ecef8ad9ca31a8372d0c353", "bar", ZoneCustomSSLOptions{})
	assert.Error(t, err)
}

func TestReprioritizeSSL(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method, "Expected method 'PUT', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"certificates":[{"ID":"5a7805061c76ada191ed06f989cc3dac","priority":2},{"ID":"9a7806061c88ada191ed06f989cc3dac","priority":1}]}`, string(b))
		}

		w.Header().Set("content-type", "application/json")
		// XXX: Test response flow properly.
		// Current response assertion uses generic example from the documentation,
		// rather than responding to the actual PUT request.
		// https://api.cloudflare.com/#custom-ssl-for-a-zone-re-prioritize-ssl-certificates
		fmt.Fprint(w, `{
          "success": true,
          "errors": [],
          "messages": [],
          "result": [
            {
              "id": "7e7b8deba8538af625850b7b2530034c",
              "hosts": [
                "example.com"
              ],
              "issuer": "GlobalSign",
              "signature": "SHA256WithRSA",
              "status": "active",
              "bundle_method": "ubiquitous",
              "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
              "uploaded_on": "2014-01-01T05:20:00Z",
              "modified_on": "2014-01-01T05:20:00Z",
              "expires_on": "2016-01-01T05:20:00Z",
              "priority": 1
            }
          ],
          "result_info": {
            "page": 1,
            "per_page": 20,
            "count": 1,
            "total_count": 2000
          }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates/prioritize", handler)

	hosts := make([]string, 1, 4)
	hosts[0] = "example.com"
	uploadedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2016-01-01T05:20:00Z")

	want := make([]ZoneCustomSSL, 1, 4)
	want[0] = ZoneCustomSSL{
		ID:           "7e7b8deba8538af625850b7b2530034c",
		Hosts:        hosts,
		Issuer:       "GlobalSign",
		Signature:    "SHA256WithRSA",
		Status:       "active",
		BundleMethod: "ubiquitous",
		ZoneID:       "023e105f4ecef8ad9ca31a8372d0c353",
		UploadedOn:   uploadedOn,
		ModifiedOn:   modifiedOn,
		ExpiresOn:    expiresOn,
		Priority:     1,
	}

	actual, err := client.ReprioritizeSSL("023e105f4ecef8ad9ca31a8372d0c353", []ZoneCustomSSLPriority{
		{ID: "5a7805061c76ada191ed06f989cc3dac", Priority: 2},
		{ID: "9a7806061c88ada191ed06f989cc3dac", Priority: 1},
	})

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestDeleteSSL(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method, "Expected method 'DELETE', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
          "id": "7e7b8deba8538af625850b7b2530034c"
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/custom_certificates/7e7b8deba8538af625850b7b2530034c", handler)

	err := client.DeleteSSL("023e105f4ecef8ad9ca31a8372d0c353", "7e7b8deba8538af625850b7b2530034c")
	assert.NoError(t, err, "Expected to successfully delete certificate ID '7e7b8deba8538af625850b7b2530034c', received error instead")

	err = client.DeleteSSL("023e105f4ecef8ad9ca31a8372d0c353", "bar")
	assert.Error(t, err, "Expected to error when attempting to delete certificate ID 'bar', did not receive error instead")
}
