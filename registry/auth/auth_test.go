// Copyright © 2018 SENETAS SECURITY PTY LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/docker/distribution/reference"
	dregistry "github.com/docker/docker/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Senetas/crypto-cli/registry"
	"github.com/Senetas/crypto-cli/registry/auth"
	"github.com/Senetas/crypto-cli/registry/httpclient"
	"github.com/Senetas/crypto-cli/registry/names"
)

const (
	imageName     = "cryptocli/alpine:test"
	invalidHeader = `Bearer realm=,service=,scope="repository:my-repo/my-alpine:pull,push"`
	validHeader   = `Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:my-repo/my-alpine:pull,push"`
	user          = "ahab"
	pass          = "hunter2"
)

func TestAuthenticator(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := ioutil.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusUnauthorized)
		}),
	)
	defer server.Close()

	ref, err := reference.ParseNormalizedNamed(imageName)
	require.NoError(err)

	repoInfo, err := dregistry.ParseRepositoryInfo(ref)
	require.NoError(err)

	creds, err := auth.NewDefaultCreds(repoInfo)
	require.NoError(err)

	tests := []struct {
		challenge string
		errMsg    string
	}{
		{
			`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:cryptocli:pull,push"`,
			"",
		},
		{
			fmt.Sprintf(
				`Bearer realm="%s",service="registry.docker.io",scope="repository:cryptocli:pull,push"`,
				server.URL,
			),
			fmt.Sprintf(
				"authentication failed with status: %d %s",
				http.StatusUnauthorized,
				http.StatusText(http.StatusUnauthorized),
			),
		},
	}

	for _, test := range tests {
		ch, err := auth.ParseChallengeHeader(test.challenge)
		if !assert.NoError(err) {
			continue
		}
		_, err = auth.NewAuthenticator(httpclient.DefaultClient, creds).Authenticate(ch)
		if err != nil && assert.EqualError(err, test.errMsg) || !assert.Equal(test.errMsg, "") {
			continue
		}
	}
}

func TestChallenger(t *testing.T) {
	require := require.New(t)

	ref, err := reference.ParseNormalizedNamed(imageName)
	require.NoError(err)

	nTRep, err := names.CastToTagged(ref)
	require.NoError(err)

	repoInfo, err := dregistry.ParseRepositoryInfo(ref)
	require.NoError(err)

	endpoint, err := registry.GetEndpoint(ref, *repoInfo)
	require.NoError(err)

	creds, err := auth.NewDefaultCreds(repoInfo)
	require.NoError(err)

	header, err := auth.ChallengeHeader(nTRep, *repoInfo, *endpoint, creds)
	require.NoError(err)

	ch, err := auth.ParseChallengeHeader(header)
	require.NoError(err)

	_, err = auth.NewAuthenticator(httpclient.DefaultClient, creds).Authenticate(ch)
	require.NoError(err)
}

func TestCreds(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	encoded := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pass)))

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := ioutil.ReadAll(r.Body)
			assert.NoError(err)
			if r.Header.Get("Authorization") == fmt.Sprintf("Basic %s", encoded) {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		}),
	)
	defer server.Close()

	creds := auth.NewCreds(user, pass)
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(err)

	creds.SetAuth(req)
	client := http.DefaultClient

	resp, err := client.Do(req)
	require.NoError(err)

	defer func() { require.NoError(resp.Body.Close()) }()

	require.Equal(http.StatusOK, resp.StatusCode)
}

func TestChallengerLoc(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		header string
		errMsg string
	}{
		{validHeader, ""},
		{invalidHeader, fmt.Sprintf("malformed challenge header: %s", invalidHeader)},
	}

	for _, test := range tests {
		_, err := auth.ParseChallengeHeader(test.header)
		if err != nil && assert.EqualError(err, test.errMsg) || !assert.Equal(test.errMsg, "") {
			continue
		}
	}
}

func TestChallengeHeader(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	server1 := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := ioutil.ReadAll(r.Body)
			assert.NoError(err)
			w.Header().Set("Www-Authenticate", "")
			w.WriteHeader(http.StatusUnauthorized)
		}),
	)
	defer server1.Close()

	server2 := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := ioutil.ReadAll(r.Body)
			assert.NoError(err)
			w.WriteHeader(http.StatusAccepted)
		}),
	)
	defer server2.Close()

	ref, err := reference.ParseNormalizedNamed(imageName)
	require.NoError(err)

	nTRep, err := names.CastToTagged(ref)
	require.NoError(err)

	repoInfo, err := dregistry.ParseRepositoryInfo(ref)
	require.NoError(err)

	endpoint, err := registry.GetEndpoint(ref, *repoInfo)
	require.NoError(err)

	creds, err := auth.NewDefaultCreds(repoInfo)
	require.NoError(err)

	server1URL, err := url.Parse(server1.URL)
	require.NoError(err)

	endpoint1 := &dregistry.APIEndpoint{
		Mirror: true,
		URL:    server1URL,
	}

	creds1 := auth.NewCreds(user, pass)

	server2URL, err := url.Parse(server2.URL)
	require.NoError(err)

	endpoint2 := &dregistry.APIEndpoint{
		Mirror: true,
		URL:    server2URL,
	}

	creds2 := auth.NewCreds(user, pass)

	tests := []struct {
		ref      reference.Named
		repoInfo dregistry.RepositoryInfo
		endpoint *dregistry.APIEndpoint
		creds    auth.Credentials
		errMsg   string
	}{
		{nTRep, *repoInfo, endpoint, creds, ""},
		{nTRep, *repoInfo, endpoint1, creds1, "login error"},
		{nTRep, *repoInfo, endpoint2, creds2, "login not supported"},
	}

	for _, test := range tests {
		_, err = auth.ChallengeHeader(test.ref, test.repoInfo, *test.endpoint, test.creds)
		_ = err != nil && assert.EqualError(err, test.errMsg) || !assert.Equal(test.errMsg, "")
	}
}

func TestToken(t *testing.T) {
	assert := assert.New(t)

	tokenStr := "this is the token"

	tests := []struct {
		tokenStr string
		errMsg   string
	}{
		{tokenStr, ""},
		{"", "malformed response from auth server"},
	}

	for _, test := range tests {
		tokenRaw := fmt.Sprintf(`{"token": "%s"}`, test.tokenStr)
		tokenJSON := bytes.NewBuffer([]byte(tokenRaw))
		token, err := auth.NewTokenFromResp(tokenJSON)
		if err != nil && assert.EqualError(err, test.errMsg) || !assert.Equal(test.errMsg, "") {
			continue
		}
		if !assert.Equal(tokenStr, token.String()) {
			continue
		}
		if !assert.False(token.Fresh()) {
			continue
		}
		req, err := http.NewRequest("GET", "http://localhost", nil)
		if !assert.NoError(err) {
			continue
		}
		auth.AddToRequest(token, req)
		assert.Equal(req.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", test.tokenStr))
	}
}
