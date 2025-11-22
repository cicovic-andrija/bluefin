package server

import (
	"os"
	"path/filepath"
	"strconv"
)

var _serverControl control

func Run() {
	trace(_control, "main: start: %s v1.2", filepath.Base(os.Args[0]))
	readEnvironment()
	buildDatabase()
	_serverControl.boot()
}

func readEnvironment() {
	const (
		modeEnvVar        = "DIVELOG_MODE"
		dbPathEnvVar      = "DIVELOG_DBFILE_PATH"
		ipHostEnvVar      = "DIVELOG_IP_HOST"
		portEnvVar        = "DIVELOG_PORT"
		privateKeyPathVar = "DIVELOG_PRIVATE_KEY_PATH"
		certPathVar       = "DIVELOG_CERT_PATH"
	)

	mode := os.Getenv(modeEnvVar)
	trace(_env, "%s = %q", modeEnvVar, mode)
	if mode == "" {
		mode = "prod"
	}

	if mode == "dev" {
		_serverControl.localAPI = true
		_serverControl.encryptedTraffic = false
		_serverControl.endpoint = "localhost:8072"
		trace(_control, "in mode %q (HTTP): endpoint will be http://%s", mode, _serverControl.endpoint)
	} else if mode == "prod" || mode == "prod-proxy-http" {
		ipHost := os.Getenv(ipHostEnvVar)
		trace(_env, "%s = %q", ipHostEnvVar, ipHost)
		if ipHost == "" {
			trace(_error, "%s is empty or undefined", ipHostEnvVar)
			os.Exit(1)
		}

		port := os.Getenv(portEnvVar)
		trace(_env, "%s = %q", portEnvVar, port)
		if port == "" {
			port = "443"
		} else {
			if num, err := strconv.Atoi(port); err != nil || num < 1 || num > 65535 {
				trace(_error, "value of %s is invalid or is not a valid TCP port number", portEnvVar)
				os.Exit(1)
			}
		}

		_serverControl.endpoint = ipHost + ":" + port

		if mode == "prod" {
			privateKeyPath := os.Getenv(privateKeyPathVar)
			trace(_env, "%s = %q", privateKeyPathVar, privateKeyPath)
			if privateKeyPath == "" {
				trace(_error, "%s is empty or undefined", privateKeyPathVar)
				os.Exit(1)
			}

			certPath := os.Getenv(certPathVar)
			trace(_env, "%s = %q", certPathVar, certPath)
			if certPath == "" {
				trace(_error, "%s is empty or undefined", certPathVar)
				os.Exit(1)
			}

			_serverControl.encryptionKeyPath = privateKeyPath
			_serverControl.publicCertPath = certPath
			_serverControl.encryptedTraffic = true
			trace(_control, "in mode %q (HTTPS): endpoint will be https://%s", mode, _serverControl.endpoint)
		} else {
			trace(_control, "in mode %q (HTTP): endpoint will be http://%s", mode, _serverControl.endpoint)
		}
	} else {
		trace(_error, "value of %s is invalid", modeEnvVar)
		os.Exit(1)
	}

	bluefin.Metadata.Source = os.Getenv(dbPathEnvVar)
	trace(_env, "%s = %q", dbPathEnvVar, bluefin.Metadata.Source)
	if bluefin.Metadata.Source == "" {
		trace(_error, "%s is empty or undefined", dbPathEnvVar)
		os.Exit(1)
	}
}
