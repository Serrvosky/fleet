package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockValidationItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}
type mockValidationError struct {
	Message string               `json:"message"`
	Errors  []mockValidationItem `json:"errors"`
}

func testGetAppConfig(t *testing.T, r *testResource) {

	req, err := http.NewRequest("GET", r.server.URL+"/api/v1/kolide/config", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.userToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var configInfo appConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&configInfo)
	require.Nil(t, err)
	require.NotNil(t, configInfo.SMTPSettings)
	config := configInfo.SMTPSettings
	assert.Equal(t, uint(465), config.SMTPPort)
	require.NotNil(t, configInfo.OrgInfo)
	assert.Equal(t, "Kolide", *configInfo.OrgInfo.OrgName)
	assert.Equal(t, "http://foo.bar/image.png", *configInfo.OrgInfo.OrgLogoURL)

}

func testModifyAppConfig(t *testing.T, r *testResource) {
	config := &kolide.AppConfig{
		KolideServerURL:        "https://foo.com",
		OrgName:                "Zip",
		OrgLogoURL:             "http://foo.bar/image.png",
		SMTPPort:               567,
		SMTPAuthenticationType: kolide.AuthTypeNone,
		SMTPServer:             "foo.com",
		SMTPEnableTLS:          true,
		SMTPVerifySSLCerts:     true,
		SMTPEnableStartTLS:     true,
	}
	payload := appConfigPayloadFromAppConfig(config)
	payload.SMTPTest = new(bool)

	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(payload)
	require.Nil(t, err)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/kolide/config", &buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.userToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var respBody appConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.Nil(t, err)
	require.NotNil(t, respBody.OrgInfo)
	assert.Equal(t, config.OrgName, *respBody.OrgInfo.OrgName)
	saved, err := r.ds.AppConfig()
	require.Nil(t, err)
	// verify email test succeeded
	assert.True(t, saved.SMTPConfigured)

}

func testModifyAppConfigWithValidationFail(t *testing.T, r *testResource) {
	config := &kolide.AppConfig{
		SMTPEnableStartTLS: false,
	}
	payload := appConfigPayloadFromAppConfig(config)
	payload.SMTPTest = new(bool)

	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(payload)
	require.Nil(t, err)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/kolide/config", &buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.userToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var validationErrors mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&validationErrors)
	require.Nil(t, err)
	require.Equal(t, 0, len(validationErrors.Errors))
	existing, err := r.ds.AppConfig()
	assert.Nil(t, err)
	assert.Equal(t, config.SMTPEnableStartTLS, existing.SMTPEnableStartTLS)
}
