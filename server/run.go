package server

import (
	"os"
	"path/filepath"
	"strconv"
)

var _control_block control

func Run() {
	trace(_control, "main: start: %s v1.3", filepath.Base(os.Args[0]))
	readEnvironment()
	runAndWaitForBuilder()
	_control_block.boot()
}

func readEnvironment() {
	const (
		modeEnvVar        = "DIVELOG_MODE"
		watchDirEnvVar    = "DIVELOG_WATCH_DIR_PATH"
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
		_control_block.localAPI = true
		_control_block.encryptedTraffic = false
		_control_block.endpoint = "localhost:8072"
		trace(_control, "in mode %q (HTTP): endpoint will be http://%s", mode, _control_block.endpoint)
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

		_control_block.endpoint = ipHost + ":" + port

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

			_control_block.encryptionKeyPath = privateKeyPath
			_control_block.publicCertPath = certPath
			_control_block.encryptedTraffic = true
			trace(_control, "in mode %q (HTTPS): endpoint will be https://%s", mode, _control_block.endpoint)
		} else {
			trace(_control, "in mode %q (HTTP): endpoint will be http://%s", mode, _control_block.endpoint)
		}
	} else {
		trace(_error, "value of %s is invalid", modeEnvVar)
		os.Exit(1)
	}

	_control_block.watchDirectoryPath = os.Getenv(watchDirEnvVar)
	trace(_env, "%s = %q", watchDirEnvVar, _control_block.watchDirectoryPath)
	if _control_block.watchDirectoryPath == "" {
		trace(_error, "%s is empty or undefined", watchDirEnvVar)
		os.Exit(1)
	}
}
