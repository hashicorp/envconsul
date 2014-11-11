package main

// Config is used to configure etcd ENV
type Config struct {
	// Etcd is the location of the etcd instance to query (may be an IP
	// address or FQDN) with port.
	Etcd string

	// Sanitize converts any "bad" characters in key values to underscores
	Sanitize bool

	// Upcase converts environment variables to uppercase
	Upcase bool

	CAFile   string
	CertFile string
	KeyFile  string
}
