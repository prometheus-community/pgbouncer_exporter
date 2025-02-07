package main

import (
	"fmt"
	"github.com/google/go-querystring/query"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type CredentialsErrorInterface interface {
	error
	SetIndex(index int)
	GetIndex() int
	GetField() string
	Unwrap() error
}

type CredentialsError struct {
	field   string
	message string
	index   int
	error   error
}

func (e *CredentialsError) Unwrap() error {
	return e.error
}

func (e *CredentialsError) Error() string {
	message := e.message
	if e.error != nil {
		message += ": " + e.error.Error()
	}

	message = fmt.Sprintf("validation failed for field %s: %s", e.field, message)
	if e.index > 0 {
		return fmt.Sprintf("%s (%s credential)", message, index2human(e.index))
	} else {
		return message
	}

}
func (e *CredentialsError) SetIndex(index int) {
	e.index = index
}
func (e *CredentialsError) GetIndex() int {
	return e.index
}
func (e *CredentialsError) GetField() string {
	return e.field
}

type Credentials struct {
	Key      string         `yaml:"key"`
	Username string         `yaml:"username"`
	Password string         `yaml:"password"`
	SSL      SSLCredentials `yaml:"ssl"`
}

type SSLCredentials struct {
	Mode        string `yaml:"mode" url:"sslmode,omitempty"`
	Cert        string `yaml:"cert" url:"sslcert,omitempty"`
	Key         string `yaml:"key" url:"sslkey,omitempty"`
	Password    string `yaml:"password" url:"sslpassword,omitempty"`
	Compression string `yaml:"compression" url:"sslcompression,omitempty"`
	Negotiation string `yaml:"negotiation" url:"sslnegotiation,omitempty"`
	CertMode    string `yaml:"cert_mode" url:"sslcertmode,omitempty"`
	RootCert    string `yaml:"root_cert" url:"sslrootcert,omitempty"`
}

func (c *SSLCredentials) validateFile(key string, file string) CredentialsErrorInterface {
	if _, err := os.Stat(file); err != nil {
		return &CredentialsError{
			field:   fmt.Sprintf("ssl.%s", key),
			message: "file error",
			error:   err,
		}
	}
	return nil
}

func (c *SSLCredentials) validateEnum(key string, value string, allowedValues []string) CredentialsErrorInterface {
	k := false
	for _, allowedValue := range allowedValues {
		if value == allowedValue {
			k = true
			continue
		}
	}
	if !k {
		return &CredentialsError{
			field: fmt.Sprintf("ssl.%s", key),
			message: fmt.Sprintf(
				"unsupported value for %s: '%s', must be one of '%s'",
				key,
				value,
				strings.Join(allowedValues, "', '"),
			),
		}
	}
	return nil
}

func (c *SSLCredentials) Validate() CredentialsErrorInterface {

	if c.Mode != "" {
		allowedModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
		err := c.validateEnum("mode", c.Mode, allowedModes)
		if err != nil {
			return err
		}
	}
	if c.CertMode != "" {
		allowedModes := []string{"disable", "allow", "require"}
		err := c.validateEnum("certmode", c.CertMode, allowedModes)
		if err != nil {
			return err
		}
	}
	if c.Negotiation != "" {
		allowedModes := []string{"postgres", "direct"}
		err := c.validateEnum("negotiation", c.Negotiation, allowedModes)
		if err != nil {
			return err
		}
	}

	if c.Cert != "" {
		if err := c.validateFile("cert", c.Cert); err != nil {
			return err
		}
	}

	if c.Key != "" {
		if err := c.validateFile("key", c.Key); err != nil {
			return err
		}
	}

	if c.RootCert != "" && c.RootCert != "system" {
		if err := c.validateFile("root_cert", c.RootCert); err != nil {
			return err
		}
	}

	return nil
}

func (c *Credentials) UpdateDSN(dsn *url.URL) {

	if c.Password == "" {
		dsn.User = url.User(c.Username)
	} else {
		dsn.User = url.UserPassword(c.Username, c.Password)
	}

	sslValues, _ := query.Values(c.SSL)
	q := dsn.Query()
	for k, v := range sslValues {
		for _, vv := range v {
			q.Set(k, vv)
		}
	}
	dsn.RawQuery = q.Encode()

}

func (c *Credentials) Validate() CredentialsErrorInterface {

	matched, err := regexp.MatchString("^[a-zA-Z0-9_-]+$", c.GetKey())
	if err != nil {
		return &CredentialsError{field: "key", message: "key is invalid", error: err}
	} else if !matched {
		return &CredentialsError{field: "key", message: fmt.Sprintf("key '%s' has invalid characters, should match /^[a-zA-Z0-9_-]+$/", c.GetKey())}
	}

	if strings.TrimSpace(c.Username) == "" {
		return &CredentialsError{field: "username", message: "username is required"}
	}

	return c.SSL.Validate()

}

func (c *Credentials) GetKey() string {

	if c.Key == "" {
		return c.Username
	}

	return c.Key
}
