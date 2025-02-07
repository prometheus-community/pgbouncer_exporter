// Copyright 2024 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langu
package main

import (
	"errors"
	"net/url"
	"testing"
)

func TestHappyCredentials(t *testing.T) {

	keyWant := "credential"
	usernameWant := "username"
	passwordWant := "password"
	//clientCertificateWant := "testdata/client.crt"

	cred := Credentials{
		Key:      keyWant,
		Username: usernameWant,
		Password: passwordWant,
	}

	err := cred.Validate()
	if err != nil {
		t.Errorf("Credential validation failed unexpected. Want: nothing, Got: %v", err)
	}

	if cred.GetKey() != keyWant {
		t.Errorf("Key does not match. Want: %v, Got: %v", keyWant, cred.GetKey())
	}
	if cred.Username != usernameWant {
		t.Errorf("Username does not match. Want: %v, Got: %v", usernameWant, cred.Username)
	}
	if cred.Password != passwordWant {
		t.Errorf("Password does not match. Want: %v, Got: %v", passwordWant, cred.Password)
	}
	/*
		if cred.ClientCertificate != clientCertificateWant {
			t.Errorf("ClientCertificate does not match. Want: %v, Got: %v", clientCertificateWant, cred.ClientCertificate)
		}

	*/

	cred = Credentials{
		Username: usernameWant,
		Password: passwordWant,
	}

	err = cred.Validate()
	if err != nil {
		t.Errorf("Credential validation failed unexpected. Want: nothing, Got: %v", err)
	}

	if cred.GetKey() != usernameWant {
		t.Errorf("Key does not match. Want: %v, Got: %v", usernameWant, cred.GetKey())
	}
	if cred.Username != usernameWant {
		t.Errorf("Username does not match. Want: %v, Got: %v", usernameWant, cred.Username)
	}
	if cred.Password != passwordWant {
		t.Errorf("Password does not match. Want: %v, Got: %v", passwordWant, cred.Password)
	}

}

func TestUnHappyCredentials(t *testing.T) {

	var errCred *CredentialsError
	var err CredentialsErrorInterface

	unhappyTests := []struct {
		wantField string
		cred      Credentials
	}{
		{
			wantField: "key",
			cred: Credentials{
				Key: "key with spaces",
			},
		},
		{
			wantField: "key",
			cred: Credentials{
				Username: "username with spaces",
			},
		},
		{
			wantField: "ssl.mode",
			cred: Credentials{
				Key:      "test",
				Username: "username",
				SSL:      SSLCredentials{Mode: "test"},
			},
		},
		{
			wantField: "ssl.cert",
			cred: Credentials{
				Key:      "test",
				Username: "username",
				SSL:      SSLCredentials{Cert: "test"},
			},
		},
	}

	for _, test := range unhappyTests {

		err = test.cred.Validate()
		if err == nil {
			t.Errorf("credential.Validate should fail with an error, no error was returned")
		} else if errors.As(err, &errCred) == false {
			t.Errorf("credential.Validate should return CredentialsError error. Got: %v", err)
		} else if err.GetField() != test.wantField {
			t.Errorf("credential.Validate failed on unexpected field. Want: %s, Got: %v", test.wantField, err.GetField())
		} else if err.Error() == "" {
			t.Errorf("credential.Validate error has empty string as Error(). Got: %v", err)
		}

	}

}

func TestUpdateDSN(t *testing.T) {

	cred := Credentials{
		Key:      "test",
		Username: "username",
		Password: "password",
		SSL:      SSLCredentials{},
	}

	sslCred := Credentials{
		Key:      "testssl",
		Username: "username",
		SSL: SSLCredentials{
			Mode:        "verify-full",
			Cert:        "testdata/client.crt",
			Key:         "testdata/client.crt",
			Password:    "password",
			Compression: "1",
			Negotiation: "postgres",
			CertMode:    "allow",
			RootCert:    "testdata/client.crt", // just pointing to an existing file, file content is not used
		},
	}

	var dsn *url.URL
	var err error
	startDSN := "postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable"
	dsn, err = url.Parse(startDSN)
	if err != nil {
		t.Errorf("Failed to parse DSN, this is a error in the test suite: %v", err)
	} else if dsn.String() != startDSN {
		t.Errorf("DSN does not match. Want: %v, Got: %v", startDSN, dsn)
	}

	wantSimple := "postgres://username:password@localhost:6543/pgbouncer?sslmode=disable"
	cred.UpdateDSN(dsn)
	if dsn.String() != wantSimple {
		t.Errorf("Updated DSN does not match. Want: %v, Got: %v", wantSimple, dsn.String())
	}

	dsn, _ = url.Parse(startDSN)
	wantSSL := "postgres://username@localhost:6543/pgbouncer?sslcert=testdata%2Fclient.crt&sslcertmode=allow&sslcompression=1&sslkey=testdata%2Fclient.crt&sslmode=verify-full&sslnegotiation=postgres&sslpassword=password&sslrootcert=testdata%2Fclient.crt"

	sslCred.UpdateDSN(dsn)
	if dsn.String() != wantSSL {
		t.Errorf("Updated DSN does not match. \nWant: %v, \nGot: %v", wantSSL, dsn.String())
	}

}

func TestIndexedCredentialError(t *testing.T) {
	err := CredentialsError{
		field:   "test",
		message: "test message",
		index:   2,
	}

	if err.GetIndex() != 2 {
		t.Errorf("GetIndex does not match. Want: %v, Got: %v", 2, err.GetIndex())
	}

	want := "validation failed for field test: test message (2nd credential)"
	if err.Error() != want {
		t.Errorf("Error does not match. \nWant: %v, \nGot: %v", want, err.Error())
	}

}
func TestWrappedCredentialError(t *testing.T) {
	err := CredentialsError{
		field:   "test",
		message: "test message",
		error:   errors.New("sub error"),
	}

	wrappedErr := err.Unwrap()
	want := "sub error"
	if wrappedErr == nil {
		t.Errorf("Unwrap should return error. Got: %v", err.error)
	} else if wrappedErr.Error() != "sub error" {
		t.Errorf("Unwrapped error does not match. Want: %v, Got: %v", want, err.Error())
	}

	want = "validation failed for field test: test message: sub error"
	if err.Error() != want {
		t.Errorf("Error does not match. \nWant: %v, \nGot: %v", want, err.Error())
	}

}
